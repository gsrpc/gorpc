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
	gslogger.Log                             // Mixin Log APIs
	sync.RWMutex                             // Mixin mutex
	name         string                      // server name
	timeout      time.Duration               // net trans timeout
	cachsize     int                         // cached size
	dispatchers  map[uint16]gorpc.Dispatcher // dispatchers
	listener     net.Listener                // listener
	builder      *gorpc.RouterBuilder        // router builder
}

// NewTCPServer create new tcp server
func NewTCPServer(builder *gorpc.RouterBuilder) *TCPServer {
	return &TCPServer{
		name:        "rpc-tcp-server",
		Log:         gslogger.Get("rpc-tcp-server"),
		timeout:     gsconfig.Seconds("gorpc.timeout", 5),
		builder:     builder,
		cachsize:    gsconfig.Int("gorpc.cachesize", 1024),
		dispatchers: make(map[uint16]gorpc.Dispatcher),
	}
}

// Register .
func (server *TCPServer) Register(dispatcher gorpc.Dispatcher) *TCPServer {

	server.Lock()
	defer server.Unlock()

	server.dispatchers[dispatcher.ID()] = dispatcher

	return server
}

// Unregister .
func (server *TCPServer) Unregister(dispatcher gorpc.Dispatcher) {
	server.Lock()
	defer server.Unlock()

	if dispatcher, ok := server.dispatchers[dispatcher.ID()]; ok && dispatcher == dispatcher {
		delete(server.dispatchers, dispatcher.ID())
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

// CacheSzie .
func (server *TCPServer) CacheSzie(size int) *TCPServer {
	server.cachsize = size
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

		server.D("accept connection(%s)", conn.RemoteAddr())

		//async handle new accept connection
		go server.newChannel(conn)
	}
}

// Close .
func (server *TCPServer) Close() {
	server.Lock()
	defer server.Unlock()

	server.D("close server(%v)", server.listener)

	if server.listener != nil {
		server.listener.Close()
		server.listener = nil
	}

}

type _TCPChannel struct {
	sync.Mutex
	gslogger.Log             // Mixin Log APIs
	gorpc.Router             // Mixin Router
	name         string      // name
	raddr        string      // remote address
	conn         net.Conn    // connection
	state        gorpc.State // connection state
}

func (server *TCPServer) newChannel(conn net.Conn) {

	channel := &_TCPChannel{
		name:  server.name,
		raddr: conn.RemoteAddr().String(),
		conn:  conn,
		state: gorpc.StateConnected,
	}

	channel.Router = server.builder.Create(channel)

	server.RLock()
	defer server.RUnlock()

	for _, dispatcher := range server.dispatchers {
		channel.Register(dispatcher)
	}

	channel.Router.StateChanged(gorpc.StateConnected)

	go channel.recvLoop(conn)

	go channel.sendLoop(conn)
}

func (channel *_TCPChannel) recvLoop(conn net.Conn) {
	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := gorpc.ReadMessage(stream)

		if err != nil {
			channel.E("[%s:%s] recv message error \n%s", channel.name, channel.raddr, err)
			channel.close(conn)
			break
		}

		channel.Router.RecvMessage(msg)
	}
}

func (channel *_TCPChannel) sendLoop(conn net.Conn) {

	stream := gorpc.NewStream(conn, conn)

	for {

		msg, ok := channel.SendQ()

		if !ok {
			break
		}

		err := gorpc.WriteMessage(stream, msg)

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
	channel.Disconnect()
}

// Disconnect .
func (channel *_TCPChannel) Disconnect() {

	channel.Lock()
	defer channel.Unlock()

	if channel.conn != nil {
		channel.conn.Close()
	}
}

func (channel *_TCPChannel) close(conn net.Conn) {

	channel.Lock()
	defer channel.Unlock()

	if conn != nil {
		conn.Close()
	}

	if channel.state != gorpc.StateConnected && channel.conn != conn {
		return
	}

	channel.Router.StateChanged(gorpc.StateDisconnect)
}
