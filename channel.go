package gorpc

// Future .
type Future interface {
	Wait() (callReturn *Response, err error)
}

// Channel .
type Channel interface {
	Post(call *Request) error
	Send(call *Request) (Future, error)
}

// MessageChannel .
type MessageChannel interface {
	SendMessage(message *Message) error
}

// ClosableChannel .
type ClosableChannel interface {
	CloseChannel()
}

// Send send method
type Send func(call *Request) (Future, error)

// Wait .
type Wait func() (callReturn *Response, err error)

// Send .
func (sendfunc Send) Send(call *Request) (Future, error) {
	return sendfunc(call)
}

// Wait .
func (waitfunc Wait) Wait() (callReturn *Response, err error) {
	return waitfunc()
}
