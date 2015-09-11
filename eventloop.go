package gorpc

import (
	"time"

	"github.com/gsrpc/gorpc/timer"
)

// EventLoop .
type EventLoop interface {
	// Execute push task into task queue
	Execute(task func()) error
	// After delay execute task
	After(delay time.Duration, task func()) *timer.Timer
}
