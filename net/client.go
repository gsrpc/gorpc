package net

import (
	"net"
	"sync"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// TCPClient gorpc client
type TCPClient struct {
	gslogger.Log                        // Mixin Log APIs
	sync.RWMutex                        // Mixin mutex
	pipeline     gorpc.Pipeline         // pipeline
	builder      *gorpc.PipelineBuilder // pipeline
	name         string                 // client name
	raddr        string                 // remote service address
	state        gorpc.State            // connect state maching
	conn         net.Conn               // connection
	retry        time.Duration          // reconnect timeout
}

// NewTCPClient create new tcp client
func NewTCPClient(name string, raddr string, builder *gorpc.PipelineBuilder) *TCPClient {
	client := &TCPClient{
		name:    name,
		Log:     gslogger.Get("rpc-tcp-client"),
		raddr:   raddr,
		state:   gorpc.StateDisconnect,
		builder: builder,
	}

	return client
}

// Connect .
func (client *TCPClient) Connect(retry time.Duration) *TCPClient {
	client.Lock()
	defer client.Unlock()

	if client.state != gorpc.StateDisconnect {
		return client
	}

	client.retry = retry

	client.state = gorpc.StateConnecting

	go func() {
		conn, err := net.Dial("tcp", client.raddr)

		if err != nil {

			client.Lock()
			client.state = gorpc.StateDisconnect
			client.Unlock()

			client.E("%s connect server error:%s", client.name, gserrors.New(err))

			if retry != 0 {
				time.AfterFunc(retry, func() {
					client.Connect(retry)
				})
			}

			return
		}

		client.connected(conn)

	}()

	return client
}

// Close .
func (client *TCPClient) Close() {

	client.Lock()
	defer client.Unlock()

	if client.conn != nil {
		client.I("close tcp connection %s(%p)", client.conn.RemoteAddr(), client.conn)
		client.conn.Close()
	}

	if client.pipeline != nil {
		client.pipeline.Close()
		client.I("close  pipeline %s(%p)", client.name, client.pipeline)
	}

	client.conn = nil

	client.pipeline = nil

	client.state = gorpc.StateClosed
}

func (client *TCPClient) close(conn net.Conn) {
	client.Lock()
	defer client.Unlock()

	if conn != nil {
		client.I("close tcp connection %s(%p)", conn.RemoteAddr(), conn)
		conn.Close()
	}

	if client.state != gorpc.StateConnected && client.conn != conn {
		return
	}

	if client.pipeline != nil {
		client.pipeline.Close()
		client.I("close  pipeline %s(%p)", client.name, client.pipeline)
	}

	client.conn = nil

	client.pipeline = nil

	if client.retry != 0 && client.state != gorpc.StateClosed {

		client.state = gorpc.StateDisconnect

		go client.Connect(client.retry)
	}

}

func (client *TCPClient) connected(conn net.Conn) {
	client.Lock()
	defer client.Unlock()

	if client.state != gorpc.StateConnecting {
		conn.Close()
		return
	}

	var err error

	client.pipeline, err = client.builder.Build(client.name)

	if err != nil {
		client.E("create pipeline error\n%s", err)

		client.state = gorpc.StateDisconnect

		if client.retry != 0 {
			go client.Connect(client.retry)
		}

		return
	}

	client.state = gorpc.StateConnected

	client.conn = conn

	go client.recvLoop(client.pipeline, conn)
	go client.sendLoop(client.pipeline, conn)

}

func (client *TCPClient) recvLoop(pipeline gorpc.Pipeline, conn net.Conn) {
	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := gorpc.ReadMessage(stream)

		if err != nil {
			client.E("%s recv message error \n%s", client.raddr, err)
			client.close(conn)
			break
		}

		err = pipeline.ChannelWrite(msg)

		if err == gorpc.ErrClosed {
			client.close(conn)
			break
		}

		if err != nil {
			client.E("pipeline write error\n%s", err)
		}
	}
}

func (client *TCPClient) sendLoop(pipeline gorpc.Pipeline, conn net.Conn) {

	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := pipeline.ChannelRead()

		if err == gorpc.ErrClosed {
			client.close(conn)
			break
		}

		if err != nil {
			client.E("pipeline write error\n%s", err)
			continue
		}

		client.V("write message[%s] :%v", msg.Code, msg.Content)

		err = gorpc.WriteMessage(stream, msg)

		gserrors.Assert(err == nil, "check WriteMessage")

		_, err = stream.Flush()

		if err != nil {
			client.close(conn)
			client.E("%s send err \n%s", client.raddr, err)
			break
		}
	}
}
