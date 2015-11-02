package gorpc

import (
	"sync"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

type handlerState int

const (
	handlerRegister handlerState = iota
	handlerActived
	handlerUnregister
)

// Context channel handler context
type Context interface {
	// current handler name
	Name() string
	// Pipeline current channel pipeline
	Pipeline() Pipeline
	// OnActive
	FireActive()
	// Send create new send pipeline message
	Send(message *Message)
	// Close close current pipeline
	Close()
}

// Handler the gorpc channel pipeline handlers
type Handler interface {
	// Register when handler had been add into one pipeline,
	// system call this function notify handler
	Register(context Context) error
	// Unregister sysm call this function when handler had been removed from
	// pipeline,you can get this pipeline object by Context#Pipeline function
	Unregister(context Context)
	// Active system call this function when pipline state trans to active
	Active(context Context) error
	// Inactive system call this function when pipeline state trans to inactive
	Inactive(context Context)
	// MessageReceived
	MessageReceived(context Context, message *Message) (*Message, error)
	// MessageSending
	MessageSending(context Context, message *Message) (*Message, error)
	// Panic handle async pipline method error
	Panic(context Context, err error)
}

// SharedHandler this handler will been shared with more than one piplines
type SharedHandler interface {
	// Lock lock this handler
	HandlerLock()
	// Unlock unlock this handler
	HandlerUnlock()
}

// HandlerF handler factory
type HandlerF func() Handler

type _Context struct {
	gslogger.Log               // mixin log
	sync.Mutex                 // context mutex
	name         string        // context bound handler name
	handler      Handler       // context bound handler
	shared       SharedHandler // shared handler
	next         *_Context     // pipeline next handler
	prev         *_Context     // pipeline prev handler
	pipeline     *_Pipeline    // pipeline which handler belongs to
	state        handlerState  // handler state
}

func newContext(name string, handler Handler, pipeline *_Pipeline, prev *_Context) (*_Context, error) {
	context := &_Context{
		Log:      gslogger.Get("handler-context"),
		name:     name,
		handler:  handler,
		pipeline: pipeline,
		prev:     prev,
		state:    handlerUnregister,
	}

	context.shared, _ = handler.(SharedHandler)

	context.lock()
	defer context.unlock()

	err := context.handler.Register(context)

	if err != nil {
		return nil, err
	}

	if context.prev != nil {
		context.prev.next = context
	}

	context.state = handlerRegister

	return context, nil

}

func (context *_Context) Close() {
	context.pipeline.onClose()
}

func (context *_Context) lock() {
	context.Lock()

	if context.shared != nil {
		context.shared.HandlerLock()
	}
}

func (context *_Context) unlock() {
	context.Unlock()

	if context.shared != nil {
		context.shared.HandlerUnlock()
	}
}

func (context *_Context) Name() string {
	return context.name
}

func (context *_Context) String() string {
	return context.name
}

func (context *_Context) Pipeline() Pipeline {
	return context.pipeline
}

func (context *_Context) FireActive() {
	context.pipeline.fireActive(context)
}

func (context *_Context) Send(message *Message) {

	context.pipeline.send(context, message)
}

func (context *_Context) onActive() (err error) {
	context.lock()

	defer func() {

		if e := recover(); e != nil {
			err = gserrors.Newf(nil, "catch unhandle error :%s", e)
		}

		context.unlock()
	}()

	if context.state != handlerRegister {
		return nil
	}

	err = context.handler.Active(context)

	if err == nil || err == ErrSkip {
		context.state = handlerActived
	}

	return
}

func (context *_Context) onPanic(err error) {
	context.lock()
	defer func() {

		if e := recover(); e != nil {
			err = gserrors.Newf(nil, "catch unhandle error :%s", e)
		}

		context.unlock()
	}()

	context.handler.Panic(context, err)
}

func (context *_Context) onInactive() {
	context.lock()
	defer func() {

		if e := recover(); e != nil {
			context.W(
				"call handler(%s) onInactive method error\n%s",
				context.Name(),
				gserrors.Newf(nil, "catch unhandle error :%s", e),
			)
		}

		context.unlock()
	}()

	if context.state != handlerActived {
		return
	}

	context.handler.Inactive(context)

	context.state = handlerRegister
}

func (context *_Context) onUnregister() {
	context.lock()
	defer func() {

		if e := recover(); e != nil {
			context.W(
				"call handler(%s) Unregister method error\n%s",
				context.Name(),
				gserrors.Newf(nil, "catch unhandle error :%s", e),
			)
		}

		context.unlock()
	}()

	if context.state == handlerUnregister {
		return
	}

	context.handler.Unregister(context)

	context.state = handlerUnregister
}

func (context *_Context) onMessageSending(message *Message) (ret *Message, err error) {
	context.lock()
	defer func() {

		if e := recover(); e != nil {
			err = gserrors.Newf(nil, "catch unhandle error :%s", e)
		}

		context.unlock()
	}()

	ret, err = context.handler.MessageSending(context, message)

	return
}

func (context *_Context) onMessageReceived(message *Message) (ret *Message, err error) {

	context.lock()

	defer func() {

		if e := recover(); e != nil {
			err = gserrors.Newf(nil, "catch unhandle error :%s", e)
		}

		context.unlock()
	}()

	ret, err = context.handler.MessageReceived(context, message)

	return
}
