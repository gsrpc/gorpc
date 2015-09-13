package tcp

import (
	"net"
	"sync"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gsevent"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

// EvtNewPipeline .
type EvtNewPipeline func(pipeline gorpc.Pipeline)

// EvtClosePipeline .
type EvtClosePipeline func(pipeline gorpc.Pipeline)

// Server server
type Server struct {
	gslogger.Log                            // Mixin Log APIs
	sync.RWMutex                            // Mixin mutex
	name             string                 // server name
	timeout          time.Duration          // net trans timeout
	listener         net.Listener           // listener
	builder          *gorpc.PipelineBuilder // router builder
	cachedsize       int                    // send q size
	eventBus         gsevent.EventBus       // events
	evtClosePipeline EvtClosePipeline       // event fire handler
	evtNewPipeline   EvtNewPipeline         // event fire handler
}

// NewServer create new tcp server
func NewServer(builder *gorpc.PipelineBuilder) *Server {
	server := &Server{
		name:       "gorpc-tcp-server",
		Log:        gslogger.Get("rpc-tcp-server"),
		timeout:    gsconfig.Seconds("gorpc.timeout", 5),
		builder:    builder,
		cachedsize: 1024,
		eventBus:   gsevent.New("gorpc-tcp-server", 1),
	}

	server.eventBus.Topic(&server.evtClosePipeline)

	server.eventBus.Topic(&server.evtNewPipeline)

	return server
}

// EvtNewPipeline .
func (server *Server) EvtNewPipeline(handler EvtNewPipeline) *Server {
	server.eventBus.Subscribe(handler)
	return server
}

// EvtClosePipeline .
func (server *Server) EvtClosePipeline(handler EvtClosePipeline) *Server {
	server.eventBus.Subscribe(handler)
	return server
}

// Name .
func (server *Server) Name(name string) *Server {
	server.name = name
	return server
}

// Timeout .
func (server *Server) Timeout(timeout time.Duration) *Server {
	server.timeout = timeout
	return server
}

// Cached .
func (server *Server) Cached(cached int) *Server {
	server.cachedsize = cached
	return server
}

func (server *Server) open(laddr string) (net.Listener, error) {
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
func (server *Server) Listen(laddr string) error {

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
func (server *Server) Close() {
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
	gslogger.Log                         // Mixin Log APIs
	pipeline         gorpc.Pipeline      // pipeline
	name             string              // name
	raddr            string              // remote address
	conn             net.Conn            // connection
	state            gorpc.State         // connection state
	cachedQ          chan *gorpc.Message // message
	closedflag       chan bool           // closed flag
	evtClosePipeline EvtClosePipeline    // closed handler
}

func (server *Server) newChannel(conn net.Conn) (err error) {

	channel := &_TCPChannel{
		Log:              gslogger.Get("gorpc-tcp-server"),
		name:             server.name,
		raddr:            conn.RemoteAddr().String(),
		conn:             conn,
		state:            gorpc.StateConnected,
		cachedQ:          make(chan *gorpc.Message, server.cachedsize),
		closedflag:       make(chan bool),
		evtClosePipeline: server.evtClosePipeline,
	}

	channel.pipeline, err = server.builder.Build(conn.RemoteAddr().String(), channel)

	if err != nil {
		return err
	}

	channel.pipeline.Active()

	server.evtNewPipeline(channel.pipeline)

	server.RLock()
	defer server.RUnlock()

	go channel.recvLoop(channel.pipeline, conn)

	go channel.sendLoop(channel.pipeline, conn)

	return nil
}

func (channel *_TCPChannel) SendMessage(message *gorpc.Message) error {

	select {
	case channel.cachedQ <- message:
	case <-channel.closedflag:
		return gorpc.ErrClosed
	}

	return nil
}

func (channel *_TCPChannel) recvLoop(pipeline gorpc.Pipeline, conn net.Conn) {
	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := gorpc.ReadMessage(stream)

		channel.V("read message[%s] :%v", msg.Code, msg.Content)

		if err != nil {
			channel.E("[%s:%s] recv message error \n%s", channel.name, channel.raddr, err)
			channel.Close()
			break
		}

		err = pipeline.Received(msg)

		if err == gorpc.ErrClosed {
			channel.Close()
			break
		}
	}
}

func (channel *_TCPChannel) sendLoop(pipeline gorpc.Pipeline, conn net.Conn) {

	stream := gorpc.NewStream(conn, conn)

	for msg := range channel.cachedQ {

		channel.V("send message %s", msg.Code)

		err := gorpc.WriteMessage(stream, msg)

		gserrors.Assert(err == nil, "check WriteMessage")

		_, err = stream.Flush()

		if err != nil {
			channel.Close()
			channel.E("[%s:%s] send err \n%s", channel.name, channel.raddr, err)
			break
		}

		channel.V("send message %s -- success", msg.Code)
	}
}

func (channel *_TCPChannel) Close() {
	channel.Lock()
	defer channel.Unlock()

	if channel.state == gorpc.StateClosed {
		return
	}

	channel.state = gorpc.StateClosed

	channel.conn.Close()

	close(channel.closedflag)

	channel.pipeline.Close()

	channel.evtClosePipeline(channel.pipeline)
}
