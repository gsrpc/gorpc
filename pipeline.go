package gorpc

import (
	"sync"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gslogger"
)

// Pipeline Channel handlers pipeline
type Pipeline interface {
	String() string
	// Name pipeline name
	Name() string
	// Close close pipeline
	Close()
	// Active trans pipeline state to active state
	Active() error
	// Inactive trans pipeline state to inactive state
	Inactive()
	// Received .
	Received(message *Message) error
	// EventLoop .
	EventLoop() EventLoop
	// Channel implement channel interface
	Channel
	// SendMessage .
	SendMessage(message *Message) error
	// AddService add new service
	AddService(dispatcher Dispatcher)
	// RemoveService remove service
	RemoveService(dispatcher Dispatcher)
	// Get Handler by name
	Handler(name string) (Handler, bool)
	// Get Sending message
	Sending() (*Message, error)
}

// PipelineBuilder pipeline builder
type PipelineBuilder struct {
	handlers   []HandlerF    // handler factories
	names      []string      // handler names
	executor   EventLoop     // event loop
	timeout    time.Duration // rpc timeout
	cachedsize int           // sending cached
}

// BuildPipeline creaet new pipeline builder
func BuildPipeline(executor EventLoop) *PipelineBuilder {
	return &PipelineBuilder{
		executor:   executor,
		timeout:    gsconfig.Seconds("gorpc.timeout", 5),
		cachedsize: gsconfig.Int("gorpc.sendQ", 1024),
	}
}

// CachedSize .
func (builder *PipelineBuilder) CachedSize(cachedsize int) *PipelineBuilder {
	builder.cachedsize = cachedsize
	return builder
}

// Handler append new handler builder
func (builder *PipelineBuilder) Handler(name string, handlerF HandlerF) *PipelineBuilder {

	builder.handlers = append(builder.handlers, handlerF)

	builder.names = append(builder.names, name)

	return builder
}

type _Pipeline struct {
	gslogger.Log                               // Mixin log APIs
	sync.Mutex                                 // pipeline sync locker
	Sink                                       // Mixin sink
	name         string                        // pipeline name
	header       *_Context                     // pipeline head handler
	tail         *_Context                     //  tail
	executor     EventLoop                     // event loop
	closedflag   chan bool                     // pipeline closed flag
	sendcached   chan func() (*Message, error) // send cache Q
}

// Build create new Pipeline
func (builder *PipelineBuilder) Build(name string) (Pipeline, error) {

	pipeline := &_Pipeline{
		Log:        gslogger.Get("pipeline"),
		name:       name,
		executor:   builder.executor,
		sendcached: make(chan func() (*Message, error), builder.cachedsize),
		closedflag: make(chan bool),
	}

	pipeline.Sink = NewSink(name, builder.executor, pipeline, builder.timeout)

	var err error

	for i, f := range builder.handlers {
		pipeline.tail, err = newContext(builder.names[i], f(), pipeline, pipeline.tail)

		if pipeline.header == nil {
			pipeline.header = pipeline.tail
		}

		if err != nil {
			return nil, err
		}
	}

	return pipeline, nil
}

func (pipeline *_Pipeline) EventLoop() EventLoop {
	return pipeline.executor
}

func (pipeline *_Pipeline) String() string {
	return pipeline.name
}

func (pipeline *_Pipeline) Name() string {
	return pipeline.name
}

func (pipeline *_Pipeline) Handler(name string) (Handler, bool) {
	current := pipeline.header

	for current != nil {
		if current.Name() == name {
			return current.handler, true
		}

		current = current.next
	}

	return nil, false
}

func (pipeline *_Pipeline) Active() error {

	current := pipeline.header

	pipeline.closedflag = make(chan bool)

	for current != nil {

		pipeline.V("%s active handler(%s)", pipeline, current)

		if err := current.onActive(); err != nil {
			if err == ErrSkip {
				return nil
			}
			return err
		}

		current = current.next
	}

	return nil
}

func (pipeline *_Pipeline) Inactive() {

	pipeline.Lock()

	select {
	case <-pipeline.closedflag:
	default:
		close(pipeline.closedflag)
	}

	pipeline.Unlock()

	current := pipeline.header

	for current != nil {

		current.onInactive()

		current = current.next
	}
}

func (pipeline *_Pipeline) Received(message *Message) error {

	current := pipeline.header

	var err error

	for current != nil {

		message, err = current.onMessageReceived(message)

		// if has error
		if err != nil {
			return err
		}

		if message == nil {
			return nil
		}

		current = current.next
	}

	if message != nil {
		return pipeline.Sink.MessageReceived(message)
	}

	return nil
}

func (pipeline *_Pipeline) Sending() (*Message, error) {

	for {
		select {
		case f := <-pipeline.sendcached:
			message, err := f()

			if err != nil {
				return nil, err
			}

			if message != nil {
				return message, nil
			}

		case <-pipeline.closedflag:
			return nil, ErrClosed
		}
	}

}

func (pipeline *_Pipeline) fireActive(context *_Context) {
	current := context.next

	for current != nil {

		if err := current.onActive(); err != nil {

			if err == ErrSkip {
				return
			}

			pipeline.E("active %s error :%s", current, err)

			context.onPanic(err)

			pipeline.onClose()

			return
		}

		pipeline.V("%s active handler(%s)", pipeline, current)

		current = current.next
	}
}

func (pipeline *_Pipeline) Close() {

	current := pipeline.header

	for current != nil {

		current.onUnregister()

		current = current.next
	}

}

func (pipeline *_Pipeline) onClose() {

	pipeline.Lock()

	select {
	case <-pipeline.closedflag:
	default:
		close(pipeline.closedflag)
	}

	pipeline.Unlock()

	current := pipeline.header

	for current != nil {

		current.onInactive()

		current = current.next
	}
}

func (pipeline *_Pipeline) onSend(f func() (*Message, error)) error {
	select {
	case pipeline.sendcached <- f:
		return nil
	default:
		return ErrOverflow
	}
}

func (pipeline *_Pipeline) send(context *_Context, message *Message) error {

	return pipeline.onSend(func() (*Message, error) {
		current := context.prev

		var err error

		for current != nil {
			message, err = current.onMessageSending(message)

			// if has error
			if err != nil {

				pipeline.W("handle sending message error\n%s", err)

				context.onPanic(err)

				return nil, err
			}

			if message == nil {
				return nil, nil
			}

			current = current.prev
		}

		return message, nil
	})
}

func (pipeline *_Pipeline) SendMessage(message *Message) error {

	return pipeline.onSend(func() (*Message, error) {
		current := pipeline.tail

		var err error

		for current != nil {
			message, err = current.onMessageSending(message)

			// if has error
			if err != nil {

				pipeline.E("pipeline(%s) send message error :%s", pipeline.name, err)

				return nil, err
			}

			if message == nil {
				return nil, nil
			}

			current = current.prev
		}

		return message, nil
	})
}
