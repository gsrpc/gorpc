package gorpc

import (
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc/timer"
)

// EventLoop .
type EventLoop interface {
	// Close close event loop
	Close()
	// Execute push task into task queue
	Execute(task func())
	// After delay execute task
	After(delay time.Duration, task func()) *timer.Timer
}

type _EventLoop struct {
	gslogger.Log              // mixin log
	timewheel    *timer.Wheel // time wheel
	taskQ        chan func()  // task fifo queue
	closed       chan bool    // close flag
}

// NewEventLoop .
func NewEventLoop(current uint32, cached uint32, timerTick time.Duration) EventLoop {
	loop := &_EventLoop{
		Log:       gslogger.Get("eventloop"),
		timewheel: timer.NewWheel(timerTick),
		taskQ:     make(chan func(), cached),
		closed:    make(chan bool),
	}

	for i := uint32(0); i < current; i++ {
		go func() {
			for task := range loop.taskQ {
				loop.execute(task)
			}
		}()
	}

	return loop

}

func (loop *_EventLoop) execute(task func()) {
	defer func() {
		if e := recover(); e != nil {
			loop.W("catched unhandle exception\n%s", gserrors.Newf(nil, "%s", e))
		}
	}()

	task()
}

func (loop *_EventLoop) Close() {
	close(loop.closed)
}

func (loop *_EventLoop) Execute(task func()) {
	select {
	case loop.taskQ <- task:
	case <-loop.closed:
	}
}

func (loop *_EventLoop) After(delay time.Duration, task func()) *timer.Timer {
	return loop.timewheel.AfterFunc(delay, task)
}
