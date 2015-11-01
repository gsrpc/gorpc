package gorpc

// Dispatcher .
type Dispatcher interface {
	Dispatch(call *Request) (callReturn *Response, err error)
	ID() uint16
	String() string
}
