package gorpc

import (
	"sync"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Context .
type Context interface {
	Close()
	// Write write message into read pipeline
	Write(message *Message)
}

// Handler gorpc handler
type Handler interface {
	OpenHandler(context Context) error
	CloseHandler(context Context)
	HandleWrite(context Context, message *Message) (*Message, error)
	HandleRead(context Context, message *Message) (*Message, error)
	HandleError(context Context, err error) error
}

// HandlerF handler create method
type HandlerF func() Handler

// Pipeline .
type Pipeline interface {
	Close()
	ChannelWrite(message *Message) error
	ChannelRead() (*Message, bool)
	AddHandler(handler Handler) Pipeline
}

// PipelineBuilder .
type PipelineBuilder struct {
	creators   []HandlerF // handler creators
	cachedsize int        // cachedsize
}

// BuildPipeline .
func BuildPipeline() *PipelineBuilder {
	return &PipelineBuilder{
		cachedsize: gsconfig.Int("gorpc.pipeline.cached", 1024),
	}
}

// CacheSize .
func (builder *PipelineBuilder) CacheSize(size int) *PipelineBuilder {
	builder.cachedsize = size
	return builder
}

// Handler .
func (builder *PipelineBuilder) Handler(f HandlerF) *PipelineBuilder {
	builder.creators = append(builder.creators, f)
	return builder
}

// Build create real pipline
func (builder *PipelineBuilder) Build(name string) (Pipeline, error) {
	pipeline := &_Pipeline{
		Log:        gslogger.Get("pipline"),
		name:       name,
		cachedsize: builder.cachedsize,
		state:      StateConnecting,
		readQ:      make(chan *Message, builder.cachedsize),
	}

	for _, createf := range builder.creators {
		pipeline.addHandler(createf())
	}

	pipeline.Lock()
	defer pipeline.Unlock()

	err := pipeline.open()

	if err == nil {
		pipeline.state = StateConnected
	}

	return pipeline, err
}

type _Handler struct {
	Handler             // Mixin handler
	Next     *_Handler  // list next node
	Prev     *_Handler  // list prev node
	pipeline *_Pipeline // pipline belongs
}

func (handler *_Handler) Close() {
	handler.pipeline.Close()
}

func (handler *_Handler) Write(message *Message) {
	handler.pipeline.write(handler, message)
}

// _Pipeline .
type _Pipeline struct {
	gslogger.Log               // Mixin log APIs
	sync.Mutex                 // Mixin mutex
	name         string        // pipline name
	header       *_Handler     // handler list header
	tail         *_Handler     // handler list tail
	cachedsize   int           // cachedsize
	state        State         // pipline state
	readQ        chan *Message // message readQ
}

func (pipeline *_Pipeline) ChannelWrite(message *Message) error {
	pipeline.Lock()
	defer pipeline.Unlock()

	if pipeline.state != StateConnected {
		pipeline.W("drop write message into write pipeline -- pipline closed")
		return ErrClosed
	}

	err := pipeline.foreach(func(handler *_Handler) (err error) {
		message, err = handler.HandleWrite(handler, message)
		return
	})

	return err
}

func (pipeline *_Pipeline) ChannelRead() (message *Message, ok bool) {
	message, ok = <-pipeline.readQ

	return
}

func (pipeline *_Pipeline) AddHandler(handler Handler) Pipeline {
	pipeline.Lock()
	defer pipeline.Unlock()

	context := pipeline.addHandler(handler)

	context.OpenHandler(context)

	return pipeline
}

func (pipeline *_Pipeline) Close() {

	pipeline.Lock()
	defer pipeline.Unlock()

	if pipeline.state != StateConnected {
		return
	}

	pipeline.state = StateClosed

	go func() {
		pipeline.foreach(func(handler *_Handler) error {
			handler.CloseHandler(handler)

			return nil
		})

		close(pipeline.readQ)
	}()
}

func (pipeline *_Pipeline) foreach(f func(handler *_Handler) error) error {

	context := pipeline.header

	for context != nil {
		err := f(context)

		if err != nil {
			return err
		}

		context = context.Next
	}

	return nil
}

func (pipeline *_Pipeline) write(handler *_Handler, message *Message) {
	go func() {
		pipeline.Lock()
		defer pipeline.Unlock()

		if pipeline.state != StateConnected {
			pipeline.W("drop write message into read pipline -- pipline closed")
			return
		}

		var err error

		context := handler.Prev

		for context != nil {

			message, err = context.HandleRead(context, message)

			if err != nil {
				pipeline.handleReadError(context, handler, err)
				return
			}

			context = context.Prev
		}

		select {
		case pipeline.readQ <- message:
		default:
			err = gserrors.Newf(ErrOverflow, "pipeline readQ overflow")
			pipeline.handleReadError(pipeline.header, handler, err)
		}

	}()
}

func (pipeline *_Pipeline) handleReadError(context *_Handler, source *_Handler, err error) {

	for context != source {

		err = context.HandleError(context, err)

		if err == nil {
			return
		}

		context = context.Next
	}

	context.HandleError(context, err)
}

func (pipeline *_Pipeline) open() error {
	return pipeline.foreach(func(handler *_Handler) error {
		return handler.OpenHandler(handler)
	})
}

func (pipeline *_Pipeline) addHandler(handler Handler) *_Handler {

	if pipeline.tail == nil {
		pipeline.header = &_Handler{
			Handler:  handler,
			pipeline: pipeline,
		}

		pipeline.tail = pipeline.header

		return pipeline.tail
	}

	pipeline.tail.Next = &_Handler{
		Handler:  handler,
		Prev:     pipeline.tail,
		pipeline: pipeline,
	}

	pipeline.tail = pipeline.tail.Next

	return pipeline.tail
}
