package tcp

import (
	"net"
	"sync"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// Client gorpc client
type Client interface {
	Close()
	Pipeline() gorpc.Pipeline
}

type _Client struct {
	gslogger.Log                // Mixin Log APIs
	sync.RWMutex                // Mixin mutex
	pipeline     gorpc.Pipeline // pipeline
	name         string         // client name
	raddr        string         // remote service address
	state        gorpc.State    // connect state maching
	conn         net.Conn       // connection
	retry        time.Duration  // reconnect timeout
	closedflag   chan bool      // closed flag
}

// ClientBuilder .
type ClientBuilder struct {
	raddr      string                 // remote address
	cachedsize int                    // send queue cached size
	builder    *gorpc.PipelineBuilder // builder
	retry      time.Duration          // reconnect timeout
}

// BuildClient .
func BuildClient(builder *gorpc.PipelineBuilder) *ClientBuilder {
	clientBuilder := &ClientBuilder{
		raddr:      gsconfig.String("gorpc.tcp.client.raddr", "127.0.0.1:13512"),
		cachedsize: gsconfig.Int("gorpc.tcp.client.cached", 128),
		builder:    builder,
		retry:      time.Duration(0),
	}

	return clientBuilder
}

// Reconnect .
func (builder *ClientBuilder) Reconnect(timeout time.Duration) *ClientBuilder {
	builder.retry = timeout
	return builder
}

// Cached .
func (builder *ClientBuilder) Cached(cached int) *ClientBuilder {
	builder.cachedsize = cached
	return builder
}

// Remote .
func (builder *ClientBuilder) Remote(raddr string) *ClientBuilder {
	builder.raddr = raddr
	return builder
}

// Connect .
func (builder *ClientBuilder) Connect(name string) (Client, error) {
	client := &_Client{
		name:       name,
		Log:        gslogger.Get("gorpc-tcp-client"),
		raddr:      builder.raddr,
		state:      gorpc.StateDisconnect,
		closedflag: make(chan bool),
		retry:      builder.retry,
	}

	var err error

	client.pipeline, err = builder.builder.Build(name)

	client.doconnect()

	return client, err
}

func (client *_Client) Pipeline() gorpc.Pipeline {
	return client.pipeline
}

func (client *_Client) doconnect() {
	client.Lock()
	defer client.Unlock()

	if client.state != gorpc.StateDisconnect {
		return
	}

	client.state = gorpc.StateConnecting

	go func() {
		conn, err := net.Dial("tcp", client.raddr)

		if err != nil {

			client.Lock()
			client.state = gorpc.StateDisconnect
			client.Unlock()

			client.E("%s connect server error:%s", client.name, gserrors.New(err))

			if client.retry != 0 {

				time.AfterFunc(client.retry, func() {
					client.doconnect()
				})
			}

			return
		}

		client.connected(conn)

	}()
}

func (client *_Client) connected(conn net.Conn) {
	client.Lock()
	defer client.Unlock()

	if client.state != gorpc.StateConnecting {
		conn.Close()
		return
	}

	client.pipeline.Active()

	client.state = gorpc.StateConnected

	client.conn = conn

	go client.recvLoop(client.pipeline, conn)
	go client.sendLoop(client.pipeline, conn)
}

func (client *_Client) Close() {
	client.Lock()
	defer client.Unlock()

	if client.state == gorpc.StateClosed {
		return
	}

	client.state = gorpc.StateClosed

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
	}

	client.pipeline.Close()
}

func (client *_Client) closeConn(conn net.Conn) {
	client.Lock()
	defer client.Unlock()

	if conn != nil {
		client.V("close tcp connection %s(%p)", conn.RemoteAddr(), conn)
		conn.Close()
	}

	if client.conn != conn {
		return
	}

	client.conn = nil

	client.pipeline.Inactive()

	if client.retry != 0 && client.state != gorpc.StateClosed {

		client.state = gorpc.StateDisconnect

		time.AfterFunc(client.retry, func() {
			client.doconnect()
		})
	}
}

func (client *_Client) recvLoop(pipeline gorpc.Pipeline, conn net.Conn) {
	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := gorpc.ReadMessage(stream)

		if err != nil {
			client.E("%s recv message error \n%s", client.raddr, err)
			client.closeConn(conn)
			break
		}

		client.V("recv message[%s] :%v", msg.Code, msg.Content)

		err = pipeline.Received(msg)

		if err == gorpc.ErrClosed {
			client.closeConn(conn)
			break
		}

		client.V("recv message %s -- success", msg.Code)

		if err != nil {
			client.E("pipeline write error\n%s", err)
		}
	}
}

func (client *_Client) sendLoop(pipeline gorpc.Pipeline, conn net.Conn) {

	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := client.pipeline.Sending()

		if err != nil {
			client.closeConn(conn)
			client.E("%s send err \n%s", client.raddr, err)
			break
		}

		client.V("write message[%s] :%v", msg.Code, msg.Content)

		err = gorpc.WriteMessage(stream, msg)

		gserrors.Assert(err == nil, "check WriteMessage")

		_, err = stream.Flush()

		if err != nil {
			client.closeConn(conn)
			client.E("%s send err \n%s", client.raddr, err)
			break
		}
	}
}
