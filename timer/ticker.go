package timer

import (
	"time"
)

// Ticker .
type Ticker struct {
	C <-chan time.Time
	r *timer
}

// NewTicker .
func NewTicker(d time.Duration) *Ticker {
	return defaultWheel.NewTicker(d)
}

// TickFunc .
func TickFunc(d time.Duration, f func()) *Ticker {
	return defaultWheel.TickFunc(d, f)
}

// Tick .
func Tick(d time.Duration) <-chan time.Time {
	return defaultWheel.Tick(d)
}

// Stop .
func (t *Ticker) Stop() {
	t.r.w.delTimer(t.r)
}

// Reset .
func (t *Ticker) Reset(d time.Duration) {
	t.r.w.resetTimer(t.r, d, d)
}
