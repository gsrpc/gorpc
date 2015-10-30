package gorpc

import (
	"bytes"
	"sync"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
)

// Sink .
type Sink interface {
	Channel
	AddService(dispatcher Dispatcher)
	RemoveService(dispatcher Dispatcher)
	MessageReceived(message *Message) error
}

type _Sink struct {
	gslogger.Log                       // Mixin Log APIs
	sync.RWMutex                       // mutex
	name         string                // name
	timeout      time.Duration         // timeout
	seqID        uint32                // sequence id
	dispatchers  map[uint16]Dispatcher // register dispatchers
	promises     map[uint32]Promise    // rpc promise
	channel      MessageChannel        // channel
	eventLoop    EventLoop             // event loop
}

// NewSink .
func NewSink(name string, eventLoop EventLoop, channel MessageChannel, timeout time.Duration) Sink {
	return &_Sink{
		Log:         gslogger.Get("sink"),
		name:        name,
		timeout:     timeout,
		dispatchers: make(map[uint16]Dispatcher),
		promises:    make(map[uint32]Promise),
		channel:     channel,
		eventLoop:   eventLoop,
	}
}

// Register .
func (sink *_Sink) AddService(dispatcher Dispatcher) {
	sink.Lock()
	defer sink.Unlock()

	sink.dispatchers[dispatcher.ID()] = dispatcher
}

// Unreigster .
func (sink *_Sink) RemoveService(dispatcher Dispatcher) {
	sink.Lock()
	defer sink.Unlock()

	if dispatcher, ok := sink.dispatchers[dispatcher.ID()]; ok && dispatcher == dispatcher {
		delete(sink.dispatchers, dispatcher.ID())
	}
}

func (sink *_Sink) Promise() (Promise, uint32) {

	sink.Lock()
	defer sink.Unlock()

	for {
		seqID := sink.seqID

		sink.seqID++

		_, ok := sink.promises[seqID]

		if ok {
			continue
		}

		promise := NewPromise(sink.eventLoop, sink.timeout, func() {
			sink.Lock()
			defer sink.Unlock()

			delete(sink.promises, seqID)
		})

		sink.promises[seqID] = promise

		return promise, seqID
	}
}

// Post .
func (sink *_Sink) Post(call *Request) error {

	var buff bytes.Buffer
	err := WriteRequest(&buff, call)
	if err != nil {
		return err
	}

	message := NewMessage()
	message.Code = CodeRequest
	message.Content = buff.Bytes()

	err = sink.channel.SendMessage(message)
	if err != nil {
		return err
	}

	sink.V("%s post request(%d:%d:%d)", sink.name, call.ID, call.Service, call.Method)

	return nil
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

	err = sink.channel.SendMessage(message)

	if err != nil {
		promise.Cancel()
		return nil, err
	}

	sink.V("%s send request(%d:%d:%d)", sink.name, call.ID, call.Service, call.Method)

	return promise, nil
}

func (sink *_Sink) dispatchResponse(response *Response) {
	sink.Lock()
	defer sink.Unlock()

	if promise, ok := sink.promises[response.ID]; ok {
		go promise.Notify(response, nil)
		delete(sink.promises, response.ID)
		return
	}

	sink.W("%s unhandle response(%d)(%d) %s", sink.name, response.ID, gserrors.Newf(nil, ""))
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

	sink.V("%s received request(%d:%d:%d)", sink.name, request.ID, request.Service, request.Method)

	if dispatcher, ok := sink.dispatch(request.Service); ok {

		response, err := dispatcher.Dispatch(request)

		if err != nil {
			sink.E("dispatch request(%d)(%d:%d) error\n%s", request.ID, request.Service, request.Method, err)
			return nil
		}

		if response == nil {
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

		sink.channel.SendMessage(message)

		sink.V("%s send response(%d)(%d)", sink.name, response.ID)

		return nil
	}

	sink.W("[%s] unhandle request(%d)(%d:%d)", sink.name, request.ID, request.Service, request.Method)

	return nil
}

func (sink *_Sink) MessageReceived(message *Message) error {

	switch message.Code {
	case CodeRequest:

		sink.dispatchRequest(message)

		return nil

	case CodeResponse:

		response, err := ReadResponse(bytes.NewBuffer(message.Content))

		if err != nil {
			sink.E("[%s] unmarshal response error\n%s", sink.name, err)
			return err
		}

		sink.dispatchResponse(response)

		return nil

	default:

		sink.W("[%s] unsupport message(%s)", sink.name, message.Code)

		return nil
	}
}
