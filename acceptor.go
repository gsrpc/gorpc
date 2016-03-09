package gorpc

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Acceptor the gorpc server channel acceptor
type Acceptor struct {
	gslogger.Log                  // mixin log APIs
	builder      *PipelineBuilder // channel builder
	timeout      time.Duration    // rpc timeout
	name         string           // acceptor name
}

// NewAcceptor create new server channel acceptor
func NewAcceptor(name string, builder *PipelineBuilder) *Acceptor {
	return &Acceptor{
		Log:     gslogger.Get(name),
		builder: builder,
		timeout: gsconfig.Seconds("timeout", 5),
		name:    name,
	}
}

// Timeout .
func (acceptor *Acceptor) Timeout(timeout time.Duration) *Acceptor {
	acceptor.timeout = timeout
	return acceptor
}

type _Channel struct {
	gslogger.Log                    // Mixin Log APIs
	pipeline     Pipeline           // pipeline
	name         string             // name
	conn         io.ReadWriteCloser // connection
	closedflag   uint32             // closedflag
}

// Accept accept new io.ReaderWriterCloser as rpc channel
func (acceptor *Acceptor) Accept(name string, conn io.ReadWriteCloser) (pipeline Pipeline, err error) {

	channel := &_Channel{
		conn: conn,
		Log:  acceptor.Log,
		name: fmt.Sprintf("%s.%s", acceptor.name, name),
	}

	pipeline, err = acceptor.builder.Build(channel.name)

	if err != nil {
		return nil, err
	}

	channel.pipeline = pipeline

	channel.pipeline.Active()

	go channel.recvLoop()

	go channel.sendLoop()

	return
}

func (channel *_Channel) recvLoop() {
	stream := NewStream(channel.conn, channel.conn)

	for {

		msg, err := ReadMessage(stream)

		channel.V("read message[%s] :%v", msg.Code, msg.Content)

		if err != nil {
			channel.E("[%s] recv message error \n%s", channel.name, err)
			channel.close()
			break
		}

		err = channel.pipeline.Received(msg)

		if err == ErrClosed {
			channel.close()
			break
		}
	}
}

func (channel *_Channel) sendLoop() {
	stream := NewStream(channel.conn, channel.conn)

	for {

		msg, err := channel.pipeline.Sending()

		if err != nil {

			channel.I(err.Error())

			if err != ErrClosed {
				channel.E("[%s] recv message error \n%s", channel.name, err)
			}

			channel.close()

			break
		}

		channel.V("send message %s", msg.Code)

		err = WriteMessage(stream, msg)

		gserrors.Assert(err == nil, "check WriteMessage")

		_, err = stream.Flush()

		if err != nil {
			channel.close()
			channel.E("[%s] recv message error \n%s", channel.name, err)
			break
		}

		channel.V("send message %s -- success", msg.Code)
	}
}

func (channel *_Channel) close() {
	if !atomic.CompareAndSwapUint32(&channel.closedflag, 0, 1) {
		return
	}

	channel.conn.Close()
	channel.pipeline.Close()
}
