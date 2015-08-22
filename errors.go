package gorpc

import "errors"

// ...
var (
	ErrRPC      = errors.New("rpc error")
	ErrOverflow = errors.New("overflow of router queue")
	ErrTimeout  = errors.New("rpc timeout")
)
