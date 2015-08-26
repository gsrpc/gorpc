package gorpc

import (
	"sync"
	"sync/atomic"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Context .
type Context interface {
	String() string

	Close()
	// write open event into write pipeline
	Open() error
	// Write write message into read pipeline
	WriteReadPipline(message *Message)
	// Write write message into write pipeline
	WriteWritePipline(message *Message)
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
	ChannelRead() (*Message, error)
}

// PipelineBuilder .
type PipelineBuilder struct {
	creators   []HandlerF // handler creators
	names      []string   // handler names
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
func (builder *PipelineBuilder) Handler(name string, f HandlerF) *PipelineBuilder {
	builder.creators = append(builder.creators, f)
	builder.names = append(builder.names, name)
	return builder
}

// Build create real pipline
func (builder *PipelineBuilder) Build(name string) (Pipeline, error) {

	pipeline := &_Pipeline{
		Log:        gslogger.Get("pipline"),
		name:       name,
		cachedsize: builder.cachedsize,
		refcounter: 1,
		readQ:      make(chan func() (*Message, error), builder.cachedsize),
		writeQ:     make(chan func() error, builder.cachedsize),
	}

	for id, createf := range builder.creators {
		pipeline.addHandler(builder.names[id], createf())
	}

	go pipeline.writeLoop()

	err := pipeline.open(pipeline.header)

	return pipeline, err
}

type _Handler struct {
	sync.Mutex            // Mixin mutex
	name       string     //handler name
	Handler               // Mixin handler
	Next       *_Handler  // list next node
	Prev       *_Handler  // list prev node
	pipeline   *_Pipeline // pipline belongs
}

func (handler *_Handler) String() string {
	return handler.name
}

func (handler *_Handler) Close() {
	handler.pipeline.Close()
}

func (handler *_Handler) WriteReadPipline(message *Message) {
	handler.pipeline.writeReadPipline(handler.Prev, message)
}

func (handler *_Handler) WriteWritePipline(message *Message) {
	handler.pipeline.writeWritePipline(handler.Next, message)
}

func (handler *_Handler) Open() error {
	return handler.pipeline.open(handler.Next)
}

// _Pipeline .
type _Pipeline struct {
	gslogger.Log                               // Mixin log APIs
	name         string                        // pipline name
	header       *_Handler                     // handler list header
	tail         *_Handler                     // handler list tail
	cachedsize   int                           // cachedsize
	readQ        chan func() (*Message, error) // message readQ
	writeQ       chan func() error             // message readQ
	refcounter   int32                         // refcounter
}

func (pipeline *_Pipeline) String() string {
	return pipeline.name
}

func (pipeline *_Pipeline) ChannelWrite(message *Message) (err error) {

	return pipeline.writeWritePipline(pipeline.header, message)

}

func (pipeline *_Pipeline) writeLoop() {

	for handleWrite := range pipeline.writeQ {
		handleWrite()
	}
}

func (pipeline *_Pipeline) ChannelRead() (*Message, error) {

	for {
		handleRead, ok := <-pipeline.readQ

		if !ok {
			return nil, ErrClosed
		}

		message, err := handleRead()

		if err == ErrSkip {
			continue
		}

		return message, err
	}

}

func (pipeline *_Pipeline) Close() {

	if err := pipeline.lock(); err != nil {
		return
	}

	defer pipeline.unlock()

	atomic.AddInt32(&pipeline.refcounter, -1)

	close(pipeline.writeQ)

	close(pipeline.readQ)

	go func() {

		pipeline.V("close handlers ...")

		pipeline.foreach(func(handler *_Handler) error {

			handler.CloseHandler(handler)

			return nil
		})

		pipeline.V("close handlers -- success ")
	}()
}

func (pipeline *_Pipeline) protectCall(handler *_Handler, f func(handler *_Handler) error) (err error) {

	defer func() {
		if e := recover(); e != nil {

			if _, ok := e.(error); ok {
				err = gserrors.Newf(e.(error), "catched pipline exception")
			} else {
				err = gserrors.Newf(ErrUnknown, "catched pipline exception :%s", e)
			}
		}

	}()

	handler.Lock()
	defer handler.Unlock()

	err = f(handler)

	return
}

func (pipeline *_Pipeline) foreach(f func(handler *_Handler) error) error {

	context := pipeline.header

	for context != nil {

		err := pipeline.protectCall(context, f)

		if err != nil {
			if err == ErrSkip {
				return nil
			}
			return err
		}

		context = context.Next
	}

	return nil
}

func (pipeline *_Pipeline) open(handler *_Handler) (err error) {

	for context := handler; context != nil; context = context.Next {

		pipeline.V("open handler(%s)", context)

		err = pipeline.protectCall(context, func(h *_Handler) error {
			return context.OpenHandler(context)
		})

		pipeline.V("open handler(%s) -- finish", context)

		if err != nil {
			if err == ErrSkip {
				err = nil
			}

			return
		}
	}

	return
}

func (pipeline *_Pipeline) writeReadPipline(handler *_Handler, message *Message) (err error) {

	defer func() {
		if e := recover(); e != nil {
			err = ErrClosed
		}
	}()

	select {
	case pipeline.readQ <- func() (*Message, error) {
		return pipeline.handleRead(handler, message)
	}:

	default:
		err := gserrors.Newf(ErrOverflow, "pipeline readQ overflow")
		pipeline.handleReadError(pipeline.header, handler, err)
	}

	return
}

func (pipeline *_Pipeline) writeWritePipline(handler *_Handler, message *Message) (err error) {

	defer func() {
		if e := recover(); e != nil {
			err = ErrClosed
		}
	}()

	select {
	case pipeline.writeQ <- func() error {
		err := pipeline.handleWrite(handler, message)
		return err
	}:

	default:
		err := gserrors.Newf(ErrOverflow, "pipeline readQ overflow")
		pipeline.handleWriteError(pipeline.header, handler, err)
	}

	return
}

func (pipeline *_Pipeline) handleWrite(handler *_Handler, message *Message) error {

	if err := pipeline.lock(); err != nil {
		return err
	}

	defer pipeline.unlock()

	var err error

	context := handler

	for context != nil {

		pipeline.V("%s handleWrite handler(%s)", pipeline.name, context)

		err = pipeline.protectCall(context, func(*_Handler) error {

			message, err = context.HandleWrite(context, message)

			if message == nil && err == nil {
				err = ErrSkip
			}

			return err
		})

		pipeline.V("%s handleWrite handler(%s) -- finish", pipeline.name, context)

		if err != nil {

			if err == ErrSkip {
				return nil
			}

			pipeline.handleWriteError(context, handler, err)
			return err
		}

		context = context.Next
	}

	return nil
}

func (pipeline *_Pipeline) handleRead(handler *_Handler, message *Message) (*Message, error) {

	if err := pipeline.lock(); err != nil {
		return nil, err
	}

	defer pipeline.unlock()

	var err error

	context := handler

	for context != nil {

		err = pipeline.protectCall(context, func(*_Handler) error {
			message, err = context.HandleRead(context, message)

			if message == nil && err == nil {
				err = ErrSkip
			}

			return err
		})

		if err != nil {
			if err == ErrSkip {
				return message, err
			}

			pipeline.handleReadError(context, handler, err)
			return nil, err
		}

		context = context.Prev
	}

	return message, nil
}

func (pipeline *_Pipeline) handleReadError(context *_Handler, source *_Handler, err error) {

	for context != source {

		pipeline.protectCall(context, func(*_Handler) error {
			err = context.HandleError(context, err)
			return nil
		})

		if err == nil {
			return
		}

		context = context.Next
	}

	context.HandleError(context, err)
}

func (pipeline *_Pipeline) handleWriteError(context *_Handler, source *_Handler, err error) {

	for context != source {

		pipeline.V("handleError handler(%s)", context)

		pipeline.protectCall(context, func(*_Handler) error {
			err = context.HandleError(context, err)
			return nil
		})

		pipeline.V("handleError handler(%s) -- finish", context)

		if err == nil {
			return
		}

		context = context.Prev
	}

	context.HandleError(context, err)
}

func (pipeline *_Pipeline) addHandler(name string, handler Handler) *_Handler {

	if pipeline.tail == nil {
		pipeline.header = &_Handler{
			Handler:  handler,
			pipeline: pipeline,
			name:     name,
		}

		pipeline.tail = pipeline.header

		return pipeline.tail
	}

	pipeline.tail.Next = &_Handler{
		Handler:  handler,
		Prev:     pipeline.tail,
		pipeline: pipeline,
		name:     name,
	}

	pipeline.tail = pipeline.tail.Next

	return pipeline.tail
}

func (pipeline *_Pipeline) lock() error {
	if atomic.AddInt32(&pipeline.refcounter, 1) > 1 {
		return nil
	}

	atomic.AddInt32(&pipeline.refcounter, -1)

	return ErrClosed
}

func (pipeline *_Pipeline) unlock() {
	atomic.AddInt32(&pipeline.refcounter, -1)
}
