package gorpc

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Router rpc router
type Router interface {
	// Mixin Channel
	Channel
	// Post post message to router
	Post(message *Message) error
	// Register register new dispatcher
	Register(dispatcher Dispatcher)
	// Unregister unregister dispatcher
	Unreigster(dispatcher Dispatcher)
	// SendQ get send Q
	SendQ() <-chan *Message
	// RecvQ get recv Q
	RecvQ() chan<- *Message
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

// _Router .
type _Router struct {
	gslogger.Log                       // Mixin Log APIs
	sync.RWMutex                       // mutex
	sendQ        chan *Message         // message sendsendQ
	recvQ        chan *Message         // message sendsendQ
	timeout      time.Duration         // timeout
	dispatchers  map[uint16]Dispatcher // register dispatchers
	promises     map[uint16]*_Promise  //  rpc promise
	seqID        uint16                // sequence id
}

// NewRouter .
func NewRouter(name string, cachesize int, timeout time.Duration) Router {
	router := &_Router{
		Log:         gslogger.Get(fmt.Sprintf("router-%s", name)),
		sendQ:       make(chan *Message, cachesize),
		recvQ:       make(chan *Message, cachesize),
		timeout:     timeout,
		dispatchers: make(map[uint16]Dispatcher),
		promises:    make(map[uint16]*_Promise),
	}

	go router.dispatchLoop()

	return router
}

func (router *_Router) newPromise() *_Promise {
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

		return promise
	}
}

func (router *_Router) Send(call *Request) (Future, error) {

	var buff bytes.Buffer

	if err := WriteRequest(&buff, call); err != nil {
		return nil, gserrors.Newf(err, "send request(%d) error", call.ID)
	}

	message := NewMessage()

	message.Code = CodeRequest

	message.Content = buff.Bytes()

	err := router.Post(message)

	if err != nil {
		return nil, err
	}

	promise := router.newPromise()

	return promise, nil
}

func (router *_Router) Post(message *Message) error {
	select {
	case router.sendQ <- message:
		return nil
	default:
		return gserrors.Newf(ErrOverflow, "router sendQ overflow")
	}
}
func (router *_Router) Register(dispatcher Dispatcher) {
	router.Lock()
	defer router.Unlock()

	router.dispatchers[dispatcher.ID()] = dispatcher
}
func (router *_Router) Unreigster(dispatcher Dispatcher) {
	router.Lock()
	defer router.Unlock()

	if dispatcher, ok := router.dispatchers[dispatcher.ID()]; ok && dispatcher == dispatcher {
		delete(router.dispatchers, dispatcher.ID())
	}
}
func (router *_Router) SendQ() <-chan *Message {
	return router.sendQ
}

func (router *_Router) RecvQ() chan<- *Message {
	return router.recvQ
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

		router.Post(message)

		return
	}

	router.W("unhandle request(%d)(%d:%d)", request.ID, request.Service, request.Method)
}

func (router *_Router) dispatchLoop() {
	for message := range router.recvQ {
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
