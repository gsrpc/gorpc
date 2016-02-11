package handler

import (
	"github.com/gsrpc/gorpc"
)

// StateF state event callback
type StateF func(pipeline gorpc.Pipeline, state gorpc.State)

type _StateHandler struct {
	f StateF // state f
}

// NewStateHandler .
func NewStateHandler(f StateF) gorpc.Handler {
	return &_StateHandler{
		f: f,
	}
}

func (handler *_StateHandler) Register(context gorpc.Context) error {
	return nil
}

func (handler *_StateHandler) Active(context gorpc.Context) error {
	handler.f(context.Pipeline(), gorpc.StateConnected)
	return nil
}

func (handler *_StateHandler) Unregister(context gorpc.Context) {
	handler.f(context.Pipeline(), gorpc.StateClosed)
}

func (handler *_StateHandler) Inactive(context gorpc.Context) {
	handler.f(context.Pipeline(), gorpc.StateDisconnect)
}

func (handler *_StateHandler) MessageReceived(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {

	return message, nil
}
func (handler *_StateHandler) MessageSending(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {
	return message, nil
}

func (handler *_StateHandler) Panic(context gorpc.Context, err error) {
}
