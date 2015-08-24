package gorpc

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Sink .
type Sink interface {
	Close()
	Disconnect()
}

// Router rpc router
type Router interface {
	fmt.Stringer

	StateChanged(state State)
	// Send send a request
	Send(call *Request) (Future, error)
	// SendMessage process send message
	SendMessage(message *Message)
	// RecvMessage process receive message
	RecvMessage(message *Message)
	// HandlerStateChanged handle fire event for state changed
	HandlerStateChanged(state State, handler Handler)
	// HandlerSend direct recv message with handler
	HandlerSend(message *Message, handler Handler)
	// HandlerRecv direct send message with handler
	HandlerRecv(message *Message, handler Handler)
	// SendQ get send Q
	SendQ() (*Message, bool)
	// Register register dispatcher
	Register(dispatcher Dispatcher)
	// Unregister unregister dispatcher
	Unregister(dispatcher Dispatcher)
	// Close router
	Close()
	// Disconnect channel
	Disconnect()
}

type _Promise struct {
	promise chan *Response
	err     error
	timer   *time.Timer
}

func newPromise(timeout time.Duration) *_Promise {
	promise := &_Promise{
		promise: make(chan *Response, 1),
	}

	promise.timer = time.AfterFunc(timeout, func() {
		promise.Timeout()
	})

	return promise
}

func (promise *_Promise) Wait() (callReturn *Response, err error) {
	return <-promise.promise, promise.err
}

func (promise *_Promise) Notify(callReturn *Response, err error) {

	promise.err = err

	promise.promise <- callReturn

	close(promise.promise)
}

func (promise *_Promise) Timeout() {
	promise.err = ErrTimeout
	close(promise.promise)
}

func (promise *_Promise) Cancel() {
	promise.err = ErrCanceled
	close(promise.promise)
}

// RouterBuilder .
type RouterBuilder struct {
	name       string        // name
	cachedsize int           // cachedsize
	timeout    time.Duration // timeout
	handlers   *Handlers     // handlers
}

// _Router .
type _Router struct {
	gslogger.Log                       // Mixin Log APIs
	sync.RWMutex                       // mutex
	name         string                // name
	timeout      time.Duration         // timeout
	seqID        uint16                // sequence id
	dispatchers  map[uint16]Dispatcher // register dispatchers
	handlers     []Handler             // register handlers
	sendQ        chan *Message         // message sendsendQ
	recvQ        chan *Message         // message sendsendQ
	promises     map[uint16]*_Promise  // rpc promise
	sink         Sink                  // sink

}

// BuildRouter .
func BuildRouter(name string) *RouterBuilder {

	return &RouterBuilder{
		name:       name,
		timeout:    gsconfig.Seconds("gorpc.timeout", 5),
		cachedsize: gsconfig.Int("gorpc.cachesize", 1024),
		handlers:   NewHandlers(),
	}
}

// Timeout .
func (builder *RouterBuilder) Timeout(timeout time.Duration) *RouterBuilder {
	builder.timeout = timeout
	return builder
}

// CacheSize .
func (builder *RouterBuilder) CacheSize(size int) *RouterBuilder {
	builder.cachedsize = size
	return builder
}

// Handler .
func (builder *RouterBuilder) Handler(f HandlerF) *RouterBuilder {
	builder.handlers.Add(f)
	return builder
}

// Create .
func (builder *RouterBuilder) Create(sink Sink) Router {

	router := &_Router{
		Log:         gslogger.Get("router"),
		name:        builder.name,
		timeout:     builder.timeout,
		handlers:    builder.handlers.Create(),
		sendQ:       make(chan *Message, builder.cachedsize),
		recvQ:       make(chan *Message, builder.cachedsize),
		dispatchers: make(map[uint16]Dispatcher),
		promises:    make(map[uint16]*_Promise),
		sink:        sink,
	}

	go router.dispatchLoop()

	return router
}

func (router *_Router) Disconnect() {
	if router.sink != nil {
		go router.sink.Close()
	}
}

func (router *_Router) String() string {
	return router.name
}

func (router *_Router) Close() {

	if router.sink != nil {
		go router.sink.Close()
	}

	go func() {
		router.Lock()
		defer router.Unlock()

		for _, handler := range router.handlers {
			handler.HandleClose(router)
		}
	}()

}

// Promise .
func (router *_Router) Promise() (Promise, uint16) {

	router.Lock()
	defer router.Unlock()

	for {
		seqID := router.seqID

		router.seqID++

		_, ok := router.promises[seqID]

		if ok {
			continue
		}

		promise := newPromise(router.timeout)

		router.promises[seqID] = promise

		return promise, seqID
	}
}

// StateChanged .
func (router *_Router) StateChanged(state State) {
	go func() {
		router.Lock()
		defer router.Unlock()

		for _, handler := range router.handlers {
			if !handler.HandleStateChanged(router, state) {
				return
			}
		}
	}()
}

// Send .
func (router *_Router) Send(call *Request) (Future, error) {

	promise, id := router.Promise()

	call.ID = id

	var buff bytes.Buffer

	err := WriteRequest(&buff, call)

	if err != nil {
		promise.Cancel()
		return nil, err
	}

	message := NewMessage()

	message.Code = CodeRequest

	message.Content = buff.Bytes()

	router.SendMessage(message)

	return promise, nil
}

// SendMessage .
func (router *_Router) SendMessage(message *Message) {

	select {
	case router.sendQ <- message:
	default:
		go router.handleSendError(gserrors.Newf(ErrOverflow, "[%] sendQ overflow", router.name))
	}
}

func (router *_Router) RecvMessage(message *Message) {

	select {
	case router.recvQ <- message:
	default:
		go router.handleRecvError(gserrors.Newf(ErrOverflow, "[%] sendQ overflow", router.name))
	}
}

// HandlerStateChanged .
func (router *_Router) HandlerStateChanged(state State, handler Handler) {
	go func() {
		router.Lock()
		defer router.Unlock()

		processing := false

		for _, current := range router.handlers {

			if current == handler {
				processing = true
				continue
			}

			if processing {
				if !current.HandleStateChanged(router, state) {
					return
				}
			}
		}
	}()
}

// HandlerSend .
func (router *_Router) HandlerSend(message *Message, handler Handler) {

	go func() {
		router.Lock()
		defer router.Unlock()

		processing := false

		var err error

		for i := len(router.handlers); i > 0; i-- {

			current := router.handlers[i-1]

			if current == handler {
				processing = true
				continue
			}

			if processing {
				message, err = current.HandleSend(router, message)

				if err != nil {
					go router.handleSendError(err)

					return
				}
			}
		}

		if message == nil {
			return
		}

		select {
		case router.sendQ <- message:
		default:
			err := gserrors.Newf(ErrOverflow, "[%] sendQ overflow", router.name)
			router.E(err.Error())
			router.handleSendError(err)
		}

	}()

}

// HandlerRecv .
func (router *_Router) HandlerRecv(message *Message, handler Handler) {

	go func() {
		router.Lock()
		defer router.Unlock()

		processing := false

		var err error

		for _, current := range router.handlers {

			if current == handler {
				processing = true
				continue
			}

			if processing {
				message, err = current.HandleSend(router, message)

				if err != nil {
					go router.handleSendError(err)

					return
				}
			}
		}

		if message == nil {
			return
		}

		select {
		case router.recvQ <- message:
		default:
			router.handleRecvError(gserrors.Newf(ErrOverflow, "[%] sendQ overflow", router.name))
		}
	}()

}

// Register .
func (router *_Router) Register(dispatcher Dispatcher) {
	router.Lock()
	defer router.Unlock()

	router.dispatchers[dispatcher.ID()] = dispatcher
}

// Unreigster .
func (router *_Router) Unregister(dispatcher Dispatcher) {
	router.Lock()
	defer router.Unlock()

	if dispatcher, ok := router.dispatchers[dispatcher.ID()]; ok && dispatcher == dispatcher {
		delete(router.dispatchers, dispatcher.ID())
	}
}

// SendQ .
func (router *_Router) SendQ() (message *Message, ok bool) {

	for {
		message, ok = <-router.sendQ

		if !ok {
			return nil, ok
		}

		message, err := router.handleWrite(message)

		if err != nil {
			router.handleSendError(err)
			continue
		}

		if message == nil {
			continue
		}

		return message, ok
	}

}

func (router *_Router) handleRecvError(err error) {
	router.Lock()
	defer router.Unlock()

	for _, handler := range router.handlers {
		err = handler.HandleError(router, err)

		if err == nil {
			return
		}
	}
}

func (router *_Router) handleSendError(err error) {
	router.Lock()
	defer router.Unlock()

	for i := len(router.handlers); i > 0; i-- {

		handler := router.handlers[i-1]

		err = handler.HandleError(router, err)

		if err == nil {
			return
		}
	}
}

func (router *_Router) handleWrite(message *Message) (*Message, error) {

	router.Lock()
	defer router.Unlock()

	var err error

	for i := len(router.handlers); i > 0; i-- {

		handler := router.handlers[i-1]

		message, err = handler.HandleSend(router, message)

		if err != nil || message == nil {
			return message, err
		}
	}

	return message, nil
}

func (router *_Router) handleRead(message *Message) (*Message, error) {

	router.Lock()
	defer router.Unlock()

	var err error

	for _, handler := range router.handlers {

		message, err = handler.HandleRecieved(router, message)

		if err != nil || message == nil {
			return message, err
		}
	}

	return message, nil
}

func (router *_Router) dispatchResponse(response *Response) {
	router.Lock()
	defer router.Unlock()

	if promise, ok := router.promises[response.ID]; ok {
		promise.Notify(response, nil)
		return
	}

	router.W("unhandle response(%d)(%d:)", response.ID, response.Service)
}

func (router *_Router) dispatchRequest(request *Request) {
	router.RLock()
	defer router.RUnlock()

	if dispatcher, ok := router.dispatchers[request.Service]; ok {

		response, err := dispatcher.Dispatch(request)

		if err != nil {
			router.E("dispatch request(%d)(%d:%d) error\n%s", request.ID, request.Service, request.Method, err)
			return
		}

		var buff bytes.Buffer

		err = WriteResponse(&buff, response)

		if err != nil {
			router.E("marshal request(%d)(%d:%d)'s response error\n%s", request.ID, request.Service, request.Method, err)
			return
		}

		message := NewMessage()

		message.Code = CodeResponse

		message.Content = buff.Bytes()

		go router.SendMessage(message)

		return
	}

	router.W("unhandle request(%d)(%d:%d)", request.ID, request.Service, request.Method)
}

func (router *_Router) dispatchLoop() {

	var err error

	for message := range router.recvQ {

		code := message.Code

		message, err = router.handleRead(message)

		if err != nil {
			router.E("handle received message(%d) error\n%s", code, err)
			router.handleSendError(err)
			continue
		}

		if message == nil {
			router.D("handles skip received message(%s)", code)
			continue
		}

		switch message.Code {
		case CodeRequest:
			request, err := ReadRequest(bytes.NewBuffer(message.Content))

			if err != nil {
				router.E("unmarshal request error\n%s", err)
				return
			}

			router.dispatchRequest(request)

		case CodeResponse:

			response, err := ReadResponse(bytes.NewBuffer(message.Content))

			if err != nil {
				router.E("unmarshal response error\n%s", err)
				return
			}

			router.dispatchResponse(response)

		default:
			router.W("unsupport message(%s)", message.Code)
		}
	}
}
