package gorpc

import "errors"

// ...
var (
	ErrRPC      = errors.New("rpc error")
	ErrOverflow = errors.New("overflow of router queue")
	ErrTimeout  = errors.New("rpc timeout")
	ErrCanceled = errors.New("rpc canceled")
	ErrClosed   = errors.New("pipeline closed")
)
