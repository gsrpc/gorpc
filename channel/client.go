package channel

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
	gslogger.Log               // Mixin Log APIs
	sync.RWMutex               // Mixin mutex
	gorpc.Router               // Mixin Router
	name         string        // client name
	raddr        string        // remote service address
	reconnect    bool          // reconnect flag
	state        gorpc.State   // connect state maching
	conn         net.Conn      // connection
	retry        time.Duration // reconnect timeout
}

// NewTCPClient create new tcp client
func NewTCPClient(raddr string, router gorpc.Router) *TCPClient {
	return &TCPClient{
		name:   raddr,
		Log:    gslogger.Get("rpc-tcp-client"),
		raddr:  raddr,
		Router: router,
		state:  gorpc.StateDisconnect,
	}
}

// Connect .
func (client *TCPClient) Connect(retry time.Duration) {
	client.Lock()
	defer client.Unlock()

	if client.state != gorpc.StateDisconnect {
		return
	}

	client.retry = retry

	client.state = gorpc.StateConnecting

	go func() {
		conn, err := net.Dial("tcp", client.raddr)

		if err != nil && client.reconnect {

			client.Lock()
			client.state = gorpc.StateDisconnect
			client.Unlock()

			client.E("connect server error\n%s", gserrors.New(err))

			if retry != 0 {
				time.AfterFunc(retry, func() {
					client.Connect(retry)
				})
			}

			return
		}

		client.connected(conn)

	}()
}

// Close .
func (client *TCPClient) Close() {
	client.close(client.conn)
}

func (client *TCPClient) close(conn net.Conn) {
	client.Lock()
	defer client.Unlock()

	if conn != nil {
		conn.Close()
	}

	if client.state != gorpc.StateConnected && client.conn != conn {
		return
	}

	go client.Router.Close()

	client.state = gorpc.StateDisconnect

	if client.reconnect {
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

	client.conn = conn

	client.state = gorpc.StateConnected

	go client.recvLoop(conn)
	go client.sendLoop(conn)
}

func (client *TCPClient) recvLoop(conn net.Conn) {
	stream := gorpc.NewStream(conn, conn)

	for {

		msg, err := gorpc.ReadMessage(stream)

		if err != nil {
			client.E("%s recv message error \n%s", client.raddr, err)
			client.close(conn)
			break
		}

		client.Router.RecvMessage(msg)
	}
}

func (client *TCPClient) sendLoop(conn net.Conn) {

	stream := gorpc.NewStream(conn, conn)

	for msg := range client.SendQ() {

		err := gorpc.WriteMessage(stream, msg)

		gserrors.Assert(err == nil, "check WriteMessage")

		_, err = stream.Flush()

		if err != nil {
			client.close(conn)
			client.E("%s send err \n%s", client.raddr, err)
			break
		}
	}
}
