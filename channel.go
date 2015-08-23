package gorpc

// Future .
type Future interface {
	Wait() (callReturn *Response, err error)
}

// Promise .
type Promise interface {
	Future
	Notify(callReturn *Response, err error)
	Timeout()
	Cancel()
}

// Channel .
type Channel interface {
	Send(call *Request) (Future, error)
}

// MessageChannel .
type MessageChannel interface {
	Channel
	SendMessage(message *Message)
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
