package net

import (
	"net"
	"sync"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// TCPServer server
type TCPServer struct {
	gslogger.Log                        // Mixin Log APIs
	sync.RWMutex                        // Mixin mutex
	name         string                 // server name
	timeout      time.Duration          // net trans timeout
	listener     net.Listener           // listener
	builder      *gorpc.PipelineBuilder // router builder
}

// NewTCPServer create new tcp server
func NewTCPServer(builder *gorpc.PipelineBuilder) *TCPServer {
	return &TCPServer{
		name:    "rpc-tcp-server",
		Log:     gslogger.Get("rpc-tcp-server"),
		timeout: gsconfig.Seconds("gorpc.timeout", 5),
		builder: builder,
	}
}

// Name .
func (server *TCPServer) Name(name string) *TCPServer {
	server.name = name
	return server
}

// Timeout .
func (server *TCPServer) Timeout(timeout time.Duration) *TCPServer {
	server.timeout = timeout
	return server
}

func (server *TCPServer) open(laddr string) (net.Listener, error) {
	server.Lock()
	defer server.Unlock()

	if server.listener != nil {
		return nil, gserrors.Newf(nil, "server already opened")
	}

	var err error

	server.listener, err = net.Listen("tcp", laddr)

	return server.listener, err
}

// Listen .
func (server *TCPServer) Listen(laddr string) error {

	listen, err := server.open(laddr)

	if err != nil {
		return gserrors.Newf(err, "start listen %s error", laddr)
	}

	for {
		conn, err := listen.Accept()

		if err != nil {
			return gserrors.Newf(err, "tcp(%s) accept error", laddr)
		}

		server.V("accept connection(%s)", conn.RemoteAddr())

		//async handle new accept connection
		go server.newChannel(conn)
	}
}

// Close .
func (server *TCPServer) Close() {
	server.Lock()
	defer server.Unlock()

	server.V("close server(%v)", server.listener)

	if server.listener != nil {
		server.listener.Close()
		server.listener = nil
	}

}

type _TCPChannel struct {
	sync.Mutex
	gslogger.Log                // Mixin Log APIs
	pipeline     gorpc.Pipeline // pipeline
	name         string         // name
	raddr        string         // remote address
	conn         net.Conn       // connection
	state        gorpc.State    // connection state
}

func (server *TCPServer) newChannel(conn net.Conn) (err error) {

	channel := &_TCPChannel{
		Log:   gslogger.Get("tcp-server"),
		name:  server.name,
		raddr: conn.RemoteAddr().String(),
		conn:  conn,
		state: gorpc.StateConnected,
	}

	channel.pipeline, err = server.builder.Build(conn.RemoteAddr().String())

	if err != nil {
		return err
	}

	server.RLock()
	defer server.RUnlock()

	go channel.recvLoop(channel.pipeline, conn)

	go channel.sendLoop(channel.pipeline, conn)

	return nil
}

func (channel *_TCPChannel) recvLoop(pipeline gorpc.Pipeline, conn net.Conn) {
	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := gorpc.ReadMessage(stream)

		channel.V("read message[%s] :%v", msg.Code, msg.Content)

		if err != nil {
			channel.E("[%s:%s] recv message error \n%s", channel.name, channel.raddr, err)
			channel.close(conn)
			break
		}

		pipeline.ChannelWrite(msg)
	}
}

func (channel *_TCPChannel) sendLoop(pipeline gorpc.Pipeline, conn net.Conn) {

	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := pipeline.ChannelRead()

		if err == gorpc.ErrClosed {
			channel.close(conn)
			break
		}

		if err != nil {
			channel.E("pipeline write error\n%s", err)
			continue
		}

		err = gorpc.WriteMessage(stream, msg)

		gserrors.Assert(err == nil, "check WriteMessage")

		_, err = stream.Flush()

		if err != nil {
			channel.close(conn)
			channel.E("[%s:%s] send err \n%s", channel.name, channel.raddr, err)
			break
		}
	}
}

func (channel *_TCPChannel) Close() {
	channel.close(channel.conn)
}

func (channel *_TCPChannel) close(conn net.Conn) {

	channel.Lock()
	defer channel.Unlock()

	if conn != nil {
		channel.I("close tcp connection %s(%p)", conn.RemoteAddr(), conn)
		conn.Close()
	}

	if channel.state != gorpc.StateConnected && channel.conn != conn {
		return
	}

	if channel.pipeline != nil {
		channel.pipeline.Close()
		channel.I("close  pipeline %s(%p)", channel.name, channel.pipeline)
	}

	channel.conn = nil

	channel.pipeline = nil
}
