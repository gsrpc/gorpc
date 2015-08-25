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

// _Sink .
type _Sink struct {
	gslogger.Log                       // Mixin Log APIs
	sync.RWMutex                       // mutex
	name         string                // name
	timeout      time.Duration         // timeout
	seqID        uint16                // sequence id
	dispatchers  map[uint16]Dispatcher // register dispatchers
	promises     map[uint16]*_Promise  // rpc promise
	cached       chan *Message         // cached message
}

// NewSink .
func NewSink(name string, timeout time.Duration, cached int) Sink {
	return &_Sink{
		Log:         gslogger.Get("sink"),
		name:        name,
		timeout:     timeout,
		dispatchers: make(map[uint16]Dispatcher),
		promises:    make(map[uint16]*_Promise),
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

		promise := newPromise(sink.timeout)

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

func (sink *_Sink) SendMessage(message *Message) error {
	select {
	case sink.cached <- message:
		return nil
	default:
		return gserrors.Newf(ErrOverflow, "sink(%s) cached overflow", sink)
	}
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
		promise.Notify(response, nil)
		return
	}

	sink.W("unhandle response(%d)(%d:)", response.ID, response.Service)
}

func (sink *_Sink) dispatchRequest(request *Request) {
	sink.RLock()
	defer sink.RUnlock()

	if dispatcher, ok := sink.dispatchers[request.Service]; ok {

		response, err := dispatcher.Dispatch(request)

		if err != nil {
			sink.E("dispatch request(%d)(%d:%d) error\n%s", request.ID, request.Service, request.Method, err)
			return
		}

		var buff bytes.Buffer

		err = WriteResponse(&buff, response)

		if err != nil {
			sink.E("marshal request(%d)(%d:%d)'s response error\n%s", request.ID, request.Service, request.Method, err)
			return
		}

		message := NewMessage()

		message.Code = CodeResponse

		message.Content = buff.Bytes()

		go sink.SendMessage(message)

		return
	}

	sink.W("[%s] unhandle request(%d)(%d:%d)", sink.name, request.ID, request.Service, request.Method)
}

func (sink *_Sink) OpenHandler(context Context) error {

	go func() {
		for message := range sink.cached {
			context.WriteReadPipline(message)
		}
	}()

	return nil
}
func (sink *_Sink) CloseHandler(context Context) {
	close(sink.cached)
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

		sink.D("[%s] request ", sink.name)

		request, err := ReadRequest(bytes.NewBuffer(message.Content))

		if err != nil {
			sink.E("[%s] unmarshal request error\n%s", sink.name, err)
			return nil, err
		}

		sink.dispatchRequest(request)

		sink.D("[%s] request -- success", sink.name)

		return nil, nil

	case CodeResponse:

		sink.D("[%s]response ", sink.name)

		response, err := ReadResponse(bytes.NewBuffer(message.Content))

		if err != nil {
			sink.E("[%s] unmarshal response error\n%s", sink.name, err)
			return nil, err
		}

		sink.dispatchResponse(response)

		sink.D("[%s] response -- success", sink.name)

		return nil, nil

	default:

		sink.W("[%s] unsupport message(%s)", sink.name, message.Code)

		return message, nil
	}

}
