package gorpc

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Client gorpc client
type Client interface {
	Close()
	Pipeline() Pipeline
}

// ClientBuilder .
type ClientBuilder struct {
	gslogger.Log                      // Mixin Log APIs
	builder          *PipelineBuilder // builder
	reconnectTimeout time.Duration    // reconnect timeout
	name             string           // name
}

// NewClientBuilder create new client builder
func NewClientBuilder(name string, builder *PipelineBuilder) *ClientBuilder {
	return &ClientBuilder{
		builder:          builder,
		reconnectTimeout: time.Duration(0),
		name:             name,
	}
}

// Reconnect .
func (builder *ClientBuilder) Reconnect(timeout time.Duration) *ClientBuilder {
	builder.reconnectTimeout = timeout
	return builder
}

// ConnectF .
type ConnectF func() (io.ReadWriteCloser, error)

type _Client struct {
	atomic.Value                   // mixin atmoic value
	gslogger.Log                   // Mixin Log APIs
	pipeline         Pipeline      // pipeline
	name             string        // client name
	f                ConnectF      // connect function
	state            uint32        // connect state
	reconnectTimeout time.Duration // reconnect timeout
}

// Build create new client
func (builder *ClientBuilder) Build(name string, F ConnectF) (Client, error) {
	client := &_Client{
		Log:   gslogger.Get(fmt.Sprintf("%s.%s", builder.name, name)),
		name:  fmt.Sprintf("%s.%s", builder.name, name),
		f:     F,
		state: uint32(StateDisconnect),
	}

	client.Log = gslogger.Get(name)

	var err error

	client.pipeline, err = builder.builder.Build(name)

	if err != nil {
		return nil, err
	}

	client.pipeline.Active()

	client.doconnect()

	return client, nil
}

func (client *_Client) Pipeline() Pipeline {
	return client.pipeline
}

func (client *_Client) Close() {

	atomic.StoreUint32(&client.state, uint32(StateClosed))

	client.pipeline.Close()
}

func (client *_Client) doconnect() {
	if !atomic.CompareAndSwapUint32(&client.state, uint32(StateDisconnect), uint32(StateConnecting)) {
		return
	}

	go func() {
		conn, err := client.f()

		if err != nil {

			if !atomic.CompareAndSwapUint32(&client.state, uint32(StateConnecting), uint32(StateDisconnect)) {
				return
			}

			client.E("%s connect server error:%s", client.name, gserrors.New(err))

			if client.reconnectTimeout != 0 {

				time.AfterFunc(client.reconnectTimeout, func() {
					client.doconnect()
				})
			}

			return
		}

		client.connected(conn)

	}()
}

func (client *_Client) connected(conn io.ReadWriteCloser) {
	if !atomic.CompareAndSwapUint32(&client.state, uint32(StateConnecting), uint32(StateConnected)) {
		return
	}

	client.pipeline.Active()

	client.Store(conn)

	go client.recvLoop(conn)
	go client.sendLoop(conn)
}

func (client *_Client) closeconn(conn io.ReadWriteCloser) {
	if client.Load().(io.ReadWriteCloser) != conn {
		return
	}

	if !atomic.CompareAndSwapUint32(&client.state, uint32(StateConnected), uint32(StateDisconnecting)) {
		if !atomic.CompareAndSwapUint32(&client.state, uint32(StateClosed), uint32(StateClosed)) {
			return
		}
	}

	conn.Close()

	if client.reconnectTimeout != 0 {

		if atomic.CompareAndSwapUint32(&client.state, uint32(StateDisconnecting), uint32(StateDisconnect)) {
			time.AfterFunc(client.reconnectTimeout, func() {
				client.doconnect()
			})
		}
	}
}

func (client *_Client) recvLoop(conn io.ReadWriteCloser) {
	stream := NewStream(conn, conn)

	for {

		msg, err := ReadMessage(stream)

		if err != nil {
			client.E("%s recv message error \n%s", client.name, err)
			client.closeconn(conn)
			break
		}

		client.V("recv message[%s] :%v", msg.Code, msg.Content)

		err = client.pipeline.Received(msg)

		if err == ErrClosed {
			client.closeconn(conn)
			break
		}

		client.V("recv message %s -- success", msg.Code)

		if err != nil {
			client.E("pipeline write error\n%s", err)
		}
	}
}

func (client *_Client) sendLoop(conn io.ReadWriteCloser) {
	stream := NewStream(conn, conn)

	for {

		msg, err := client.pipeline.Sending()

		if err != nil {
			client.closeconn(conn)
			client.E("%s send err \n%s", client.name, err)
			break
		}

		client.V("write message[%s] :%v", msg.Code, msg.Content)

		err = WriteMessage(stream, msg)

		gserrors.Assert(err == nil, "check WriteMessage")

		_, err = stream.Flush()

		if err != nil {
			client.closeconn(conn)
			client.E("%s send err \n%s", client.name, err)
			break
		}
	}
}
