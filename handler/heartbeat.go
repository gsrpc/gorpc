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
		exitflag:  make(chan bool),
		timestamp: time.Now(),
	}
}

func (handler *_HeartbeatHandler) Register(context gorpc.Context) error {
	return nil
}

func (handler *_HeartbeatHandler) Active(context gorpc.Context) error {

	handler.context = context

	if handler.timeout != 0 {
		go handler.timeoutLoop()
	}

	return nil
}

func (handler *_HeartbeatHandler) Unregister(context gorpc.Context) {
}

func (handler *_HeartbeatHandler) Inactive(context gorpc.Context) {
	close(handler.exitflag)
}

func (handler *_HeartbeatHandler) timeoutLoop() {

	ticker := time.NewTicker(handler.timeout)

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

		case <-handler.exitflag:
			return
		}
	}
}

func (handler *_HeartbeatHandler) MessageReceived(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {

	if message.Code == gorpc.CodeHeartbeat {

		handler.V("%s recv heartbeat message", handler.context.Pipeline())

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
