package gorpc

import (
	"bytes"
	"sync"
	"time"

	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Router rpc router
type Router interface {
	Send(call *Request) (Future, error)

	SendMessage(message *Message)

	RecvMessage(message *Message)

	// HandlerSend direct recv message with handler
	HandlerSend(message *Message, handler Handler)
	// HandlerRecv direct send message with handler
	HandlerRecv(message *Message, handler Handler)
	// SendQ get send Q
	SendQ() <-chan *Message
	// RecvQ get recv Q
	RecvQ() chan<- *Message
	// Register register dispatcher
	Register(dispatcher Dispatcher)
	// Unregister unregister dispatcher
	Unregister(dispatcher Dispatcher)
	// Close router
	Close()
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
	gslogger.Log                         // Mixin Log APIs
	sync.RWMutex                         // mutex
	*RouterBuilder                       // mixin builder
	seqID          uint16                // sequence id
	dispatchers    map[uint16]Dispatcher // register dispatchers
	handlers       []Handler             // register handlers
	sendQ          chan *Message         // message sendsendQ
	recvQ          chan *Message         // message sendsendQ
	promises       map[uint16]*_Promise  // rpc promise

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
func (builder *RouterBuilder) Create() Router {

	router := &_Router{
		Log:         gslogger.Get("router"),
		handlers:    builder.handlers.Create(),
		sendQ:       make(chan *Message, builder.cachedsize),
		recvQ:       make(chan *Message, builder.cachedsize),
		dispatchers: make(map[uint16]Dispatcher),
		promises:    make(map[uint16]*_Promise),
	}

	go router.dispatchLoop()

	return router
}

func (router *_Router) Close() {
	router.Lock()
	defer router.Unlock()

	for _, handler := range router.handlers {
		handler.HandleClose(router)
	}
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

	message.Content = buff.Bytes()

	router.SendMessage(message)

	return promise, nil
}

// SendMessage .
func (router *_Router) SendMessage(message *Message) {
	var err error

	message, err = router.handleWrite(message)

	if err != nil {

		router.handleSendError(err)

		return
	}

	if message == nil {
		return
	}

	select {
	case router.sendQ <- message:
	default:
		router.handleSendError(gserrors.Newf(ErrOverflow, "[%] sendQ overflow", router.name))
	}
}

func (router *_Router) RecvMessage(message *Message) {
	var err error

	message, err = router.handleRead(message)

	if err != nil {

		router.handleSendError(err)

		return
	}

	if message == nil {
		return
	}

	select {
	case router.recvQ <- message:
	default:
		router.handleRecvError(gserrors.Newf(ErrOverflow, "[%] sendQ overflow", router.name))
	}
}

// HandlerSend .
func (router *_Router) HandlerSend(message *Message, handler Handler) {

	router.Lock()
	defer router.Unlock()

	processing := false

	var err error

	for i := len(router.handlers); i > 0; i-- {

		current := router.handlers[i-1]

		if current == handler {
			processing = true
		}

		if processing {
			message, err = current.HandleSend(router, message)

			if err != nil {
				go router.handleSendError(err)

				return
			}
		}
	}

}

// HandlerRecv .
func (router *_Router) HandlerRecv(message *Message, handler Handler) {
	router.Lock()
	defer router.Unlock()

	processing := false

	var err error

	for _, current := range router.handlers {

		if current == handler {
			processing = true
		}

		if processing {
			message, err = current.HandleSend(router, message)

			if err != nil {
				go router.handleSendError(err)

				return
			}
		}
	}
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
func (router *_Router) SendQ() <-chan *Message {
	return router.sendQ
}

// RecvQ .
func (router *_Router) RecvQ() chan<- *Message {
	return router.recvQ
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
