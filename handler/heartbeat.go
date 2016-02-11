package handler

import (
	"time"

	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

type _HeartbeatHandler struct {
	gslogger.Log               // Mixin log APIs
	timeout      time.Duration // timeout
	context      gorpc.Context // handler context
	exitflag     chan bool     // exit flag
	flag         uint32        // flag
	timestamp    time.Time     // last received heartbeat timestamp
}

// NewHeartbeatHandler create new heartbeat handler
func NewHeartbeatHandler(timeout time.Duration) gorpc.Handler {
	return &_HeartbeatHandler{
		Log:       gslogger.Get("heartbeat"),
		timeout:   timeout,
		timestamp: time.Now(),
	}
}

func (handler *_HeartbeatHandler) Register(context gorpc.Context) error {
	return nil
}

func (handler *_HeartbeatHandler) Active(context gorpc.Context) error {

	if handler.exitflag == nil {

		handler.context = context

		handler.exitflag = make(chan bool)

		handler.timestamp = time.Now()

		if handler.timeout != 0 {
			go handler.timeoutLoop(context, handler.exitflag)
		}
	}

	return nil
}

func (handler *_HeartbeatHandler) Unregister(context gorpc.Context) {
}

func (handler *_HeartbeatHandler) Inactive(context gorpc.Context) {

	if handler.exitflag != nil {

		close(handler.exitflag)

		handler.exitflag = nil
	}

}

func (handler *_HeartbeatHandler) timeoutLoop(context gorpc.Context, exitflag chan bool) {

	wheel := context.Pipeline().EventLoop().TimeWheel()

	ticker := wheel.NewTicker(handler.timeout)

	defer ticker.Stop()

	for {

		select {
		case <-ticker.C:

			if time.Now().Sub(handler.timestamp) > handler.timeout*2 {
				handler.context.Close()
				handler.W("heartbeat timeout(%s), close current pipeline(%s)", handler.timeout*2, handler.context.Pipeline())
				return
			}

			message := gorpc.NewMessage()

			message.Code = gorpc.CodeHeartbeat

			handler.context.Send(message)

			handler.V("%s send heartbeat message", handler.context.Pipeline())

		case <-exitflag:
			handler.V("exit heartbeat loop .....................")
			return
		}
	}
}

func (handler *_HeartbeatHandler) MessageReceived(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {

	if message.Code == gorpc.CodeHeartbeat {

		handler.V("%s recv heartbeat message", context.Pipeline())

		if handler.timeout != 0 {
			handler.timestamp = time.Now()
		}

		return nil, nil
	}

	return message, nil
}
func (handler *_HeartbeatHandler) MessageSending(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {
	return message, nil
}

func (handler *_HeartbeatHandler) Panic(context gorpc.Context, err error) {

}
