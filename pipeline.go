package gorpc

import (
	"sync"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
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
	// AddService add new service
	AddService(dispatcher Dispatcher)
	// RemoveService remove service
	RemoveService(dispatcher Dispatcher)
}

// PipelineBuilder pipeline builder
type PipelineBuilder struct {
	handlers []HandlerF    // handler factories
	names    []string      // handler names
	executor EventLoop     // event loop
	timeout  time.Duration // rpc timeout
}

// BuildPipeline creaet new pipeline builder
func BuildPipeline(executor EventLoop) *PipelineBuilder {
	return &PipelineBuilder{
		executor: executor,
		timeout:  gsconfig.Seconds("gorpc.timeout", 5),
	}
}

// Handler append new handler builder
func (builder *PipelineBuilder) Handler(name string, handlerF HandlerF) *PipelineBuilder {

	builder.handlers = append(builder.handlers, handlerF)

	builder.names = append(builder.names, name)

	return builder
}

type _Pipeline struct {
	gslogger.Log                // Mixin log APIs
	sync.Mutex                  // pipeline sync locker
	Sink                        // Mixin sink
	name         string         // pipeline name
	header       *_Context      // pipeline head handler
	tail         *_Context      //  tail
	executor     EventLoop      // event loop
	closedflag   bool           // pipeline closed flag
	channel      MessageChannel // message channel
}

// Build create new Pipeline
func (builder *PipelineBuilder) Build(name string, channel MessageChannel) (Pipeline, error) {

	pipeline := &_Pipeline{
		Log:      gslogger.Get("pipeline"),
		name:     name,
		executor: builder.executor,
		channel:  channel,
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
	return pipeline.EventLoop()
}

func (pipeline *_Pipeline) String() string {
	return pipeline.name
}

func (pipeline *_Pipeline) Name() string {
	return pipeline.name
}

func (pipeline *_Pipeline) Close() {
	pipeline.executor.Execute(func() {

		pipeline.Lock()
		defer pipeline.Unlock()

		if pipeline.closedflag {
			return
		}

		pipeline.closedflag = true

		current := pipeline.header

		for current != nil {

			current.onInactive()

			current.onUnregister()

			current = current.next
		}

		pipeline.channel.Close()
	})

}

func (pipeline *_Pipeline) Active() error {
	pipeline.Lock()
	defer pipeline.Unlock()

	current := pipeline.header

	for current != nil {

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
	defer pipeline.Unlock()

	current := pipeline.header

	for current != nil {

		current.onInactive()

		current = current.next
	}
}

func (pipeline *_Pipeline) Received(message *Message) error {
	pipeline.Lock()
	defer pipeline.Unlock()

	if pipeline.closedflag {
		return ErrClosed
	}

	current := pipeline.header

	for current != nil {

		message, err := current.onMessageReceived(message)

		// if has error
		if err != nil {
			return err
		}

		if message == nil {
			return nil
		}

		current = current.next
	}

	return nil
}

func (pipeline *_Pipeline) SendMessage(message *Message) error {

	pipeline.Lock()
	defer pipeline.Unlock()

	if pipeline.closedflag {
		return gserrors.Newf(ErrClosed, "pipeline(%s) closed", pipeline.name)
	}

	current := pipeline.tail

	var err error

	for current != nil {
		message, err = current.onMessageSending(message)

		// if has error
		if err != nil {
			return err
		}

		if message == nil {
			return nil
		}

		current = current.prev
	}

	if message != nil {
		return pipeline.channel.SendMessage(message)
	}

	return nil
}

func (pipeline *_Pipeline) fireActive(context *_Context) {
	pipeline.executor.Execute(func() {

		pipeline.Lock()
		defer pipeline.Unlock()

		if pipeline.closedflag {
			context.onPanic(ErrClosed)
			return
		}

		if current := context.next; current != nil {
			if err := current.onActive(); err != nil {

				if err == ErrSkip {
					return
				}

				context.onPanic(err)

				pipeline.close()

				return
			}
		}
	})
}

func (pipeline *_Pipeline) close() {

	if pipeline.closedflag {
		return
	}

	pipeline.closedflag = true

	current := pipeline.header

	for current != nil {

		current.onInactive()

		current.onUnregister()

		current = current.next
	}

	pipeline.channel.Close()
}

func (pipeline *_Pipeline) send(context *_Context, message *Message) {

	pipeline.executor.Execute(func() {

		pipeline.Lock()
		defer pipeline.Unlock()

		if pipeline.closedflag {
			context.onPanic(ErrClosed)
			return
		}

		current := context.prev

		var err error

		for current != nil {
			message, err = current.onMessageSending(message)

			// if has error
			if err != nil {
				context.onPanic(err)

				return
			}

			if message == nil {
				return
			}

			current = current.prev
		}

		if message != nil {
			err := pipeline.channel.SendMessage(message)
			if err != nil {
				pipeline.E("%s send message(%p) error :%s", err)
				context.onPanic(err)
			}
		}
	})
}
