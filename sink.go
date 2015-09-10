package gorpc

import (
	"bytes"
	"sync"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Sink rpc sink
type Sink interface {
	Handler
	MessageChannel
	Unregister(dispatcher Dispatcher)
	Register(dispatcher Dispatcher)
}

type _Promise struct {
	promise chan *Response
	err     error
	timer   *time.Timer
}

// Promise .
type Promise interface {
	Wait() (callReturn *Response, err error)
	Notify(callReturn *Response, err error)
	Timeout()
	Cancel()
}

// NewPromise .
func NewPromise(timeout time.Duration, f func()) Promise {
	promise := &_Promise{
		promise: make(chan *Response, 1),
	}

	promise.timer = time.AfterFunc(timeout, func() {
		f()
		promise.Timeout()

	})

	return promise
}

func (promise *_Promise) Wait() (callReturn *Response, err error) {
	return <-promise.promise, promise.err
}

func (promise *_Promise) Notify(callReturn *Response, err error) {

	promise.timer.Stop()

	promise.err = err

	promise.promise <- callReturn
}

func (promise *_Promise) Timeout() {
	promise.err = ErrTimeout
	promise.promise <- nil
}

func (promise *_Promise) Cancel() {
	promise.err = ErrCanceled
	close(promise.promise)
}

// _Sink .
type _Sink struct {
	gslogger.Log                       // Mixin Log APIs
	sync.RWMutex                       // mutex
	name         string                // name
	timeout      time.Duration         // timeout
	seqID        uint16                // sequence id
	dispatchers  map[uint16]Dispatcher // register dispatchers
	promises     map[uint16]Promise    // rpc promise
	cached       chan *Message         // cached message
	closed       chan bool             // closed
}

// NewSink .
func NewSink(name string, timeout time.Duration, cached int) Sink {
	return &_Sink{
		Log:         gslogger.Get("sink"),
		name:        name,
		timeout:     timeout,
		dispatchers: make(map[uint16]Dispatcher),
		promises:    make(map[uint16]Promise),
		cached:      make(chan *Message, cached),
	}
}

func (sink *_Sink) String() string {
	return sink.name
}

// Promise .
func (sink *_Sink) Promise() (Promise, uint16) {

	sink.Lock()
	defer sink.Unlock()

	for {
		seqID := sink.seqID

		sink.seqID++

		_, ok := sink.promises[seqID]

		if ok {
			continue
		}

		promise := NewPromise(sink.timeout, func() {
			sink.Lock()
			defer sink.Unlock()

			delete(sink.promises, seqID)
		})

		sink.promises[seqID] = promise

		return promise, seqID
	}
}

// Send .
func (sink *_Sink) Send(call *Request) (Future, error) {

	promise, id := sink.Promise()

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

	err = sink.SendMessage(message)

	if err != nil {
		promise.Cancel()
		return nil, err
	}

	return promise, nil
}

func (sink *_Sink) SendMessage(message *Message) (err error) {

	defer func() {
		if e := recover(); e != nil {
			err = ErrClosed
		}
	}()

	select {
	case sink.cached <- message:
	default:
		err = gserrors.Newf(ErrOverflow, "sink(%s) cached overflow", sink)
	}

	return
}

// Register .
func (sink *_Sink) Register(dispatcher Dispatcher) {
	sink.Lock()
	defer sink.Unlock()

	sink.dispatchers[dispatcher.ID()] = dispatcher
}

// Unreigster .
func (sink *_Sink) Unregister(dispatcher Dispatcher) {
	sink.Lock()
	defer sink.Unlock()

	if dispatcher, ok := sink.dispatchers[dispatcher.ID()]; ok && dispatcher == dispatcher {
		delete(sink.dispatchers, dispatcher.ID())
	}
}

func (sink *_Sink) dispatchResponse(response *Response) {
	sink.Lock()
	defer sink.Unlock()

	if promise, ok := sink.promises[response.ID]; ok {
		go promise.Notify(response, nil)
		delete(sink.promises, response.ID)
		return
	}

	sink.W("unhandle response(%d)(%d)", response.ID, response.Service)
}

func (sink *_Sink) dispatch(id uint16) (dispatcher Dispatcher, ok bool) {
	sink.RLock()
	defer sink.RUnlock()

	dispatcher, ok = sink.dispatchers[id]

	return
}

func (sink *_Sink) dispatchRequest(message *Message) error {

	request, err := ReadRequest(bytes.NewBuffer(message.Content))

	if err != nil {
		sink.E("[%s] unmarshal request error\n%s", sink.name, err)
		return err
	}

	if dispatcher, ok := sink.dispatch(request.Service); ok {

		response, err := dispatcher.Dispatch(request)

		if err != nil {
			sink.E("dispatch request(%d)(%d:%d) error\n%s", request.ID, request.Service, request.Method, err)
			return nil
		}

		var buff bytes.Buffer

		err = WriteResponse(&buff, response)

		if err != nil {
			sink.E("marshal request(%d)(%d:%d)'s response error\n%s", request.ID, request.Service, request.Method, err)
			return nil
		}

		message.Code = CodeResponse

		message.Content = buff.Bytes()

		go sink.SendMessage(message)

		return nil
	}

	sink.W("[%s] unhandle request(%d)(%d:%d)", sink.name, request.ID, request.Service, request.Method)

	return nil
}

func (sink *_Sink) OpenHandler(context Context) error {

	closed := make(chan bool)

	sink.closed = closed

	go func() {
		for {
			select {
			case message := <-sink.cached:
				context.WriteReadPipline(message)
			case <-closed:
				return
			}
		}

	}()

	return nil
}

func (sink *_Sink) CloseHandler(context Context) {
	if sink.closed != nil {
		close(sink.closed)
		sink.closed = nil
	}
}

func (sink *_Sink) HandleError(context Context, err error) error {
	return err
}

func (sink *_Sink) HandleRead(context Context, message *Message) (*Message, error) {
	return message, nil
}

func (sink *_Sink) HandleWrite(context Context, message *Message) (*Message, error) {

	switch message.Code {
	case CodeRequest:

		sink.dispatchRequest(message)

		return nil, nil

	case CodeResponse:

		response, err := ReadResponse(bytes.NewBuffer(message.Content))

		if err != nil {
			sink.E("[%s] unmarshal response error\n%s", sink.name, err)
			return nil, err
		}

		sink.dispatchResponse(response)

		return nil, nil

	default:

		sink.W("[%s] unsupport message(%s)", sink.name, message.Code)

		return message, nil
	}

}
