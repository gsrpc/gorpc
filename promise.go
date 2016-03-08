package gorpc

import (
	"time"

	"github.com/gsrpc/gorpc/timer"
)

type _Promise struct {
	promise chan *Response
	err     error
	timer   *timer.Timer
}

// Promise .
type Promise interface {
	Wait() (callReturn *Response, err error)
	Notify(callReturn *Response, err error)
	Timeout()
	Cancel()
}

// NewPromise .
func NewPromise(timewheel *timer.Wheel, timeout time.Duration, f func()) Promise {
	promise := &_Promise{
		promise: make(chan *Response, 1),
	}

	promise.timer = timewheel.AfterFunc(timeout, func() {
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
	promise.timer.Stop()

	promise.err = ErrTimeout
	promise.promise <- nil
}

func (promise *_Promise) Cancel() {
	promise.timer.Stop()

	promise.err = ErrCanceled
	promise.promise <- nil
}
