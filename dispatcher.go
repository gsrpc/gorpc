package gorpc

import "com/gsrpc"

// Dispatcher .
type Dispatcher interface {
	Dispatch(call *gsrpc.Request) (callReturn *gsrpc.Response, err error)
}

// Future .
type Future interface {
	Wait() (callReturn *gsrpc.Response, err error)
}

// Channel .
type Channel interface {
	Send(call *gsrpc.Request) Future
}
