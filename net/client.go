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
	gslogger.Log               // Mixin Log APIs
	sync.RWMutex               // Mixin mutex
	gorpc.Router               // Mixin Router
	name         string        // client name
	raddr        string        // remote service address
	state        gorpc.State   // connect state maching
	conn         net.Conn      // connection
	retry        time.Duration // reconnect timeout
}

// NewTCPClient create new tcp client
func NewTCPClient(raddr string, builder *gorpc.RouterBuilder) *TCPClient {
	client := &TCPClient{
		name:  raddr,
		Log:   gslogger.Get("rpc-tcp-client"),
		raddr: raddr,
		state: gorpc.StateDisconnect,
	}

	client.Router = builder.Create(client)

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

	client.Router.StateChanged(client.state)

	go func() {
		conn, err := net.Dial("tcp", client.raddr)

		if err != nil {

			client.Lock()
			client.state = gorpc.StateDisconnect
			client.Unlock()

			client.Router.StateChanged(client.state)

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

	return client
}

// Close .
func (client *TCPClient) Close() {

	client.D("close client")

	client.Disconnect()
}

// Disconnect .
func (client *TCPClient) Disconnect() {
	client.Lock()
	defer client.Unlock()

	if client.conn != nil {
		client.conn.Close()
	}

	client.state = gorpc.StateDisconnect
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

	client.state = gorpc.StateDisconnect

	client.Router.StateChanged(client.state)

	if client.retry != 0 {
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

	client.state = gorpc.StateConnected

	client.conn = conn

	go client.recvLoop(conn)
	go client.sendLoop(conn)

	client.Router.StateChanged(client.state)
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

	for {

		msg, ok := client.SendQ()

		if !ok {
			break
		}

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
