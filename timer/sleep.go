package timer

import "time"

// Timer .
type Timer struct {
	C <-chan time.Time
	r *timer
}

// After .
func After(d time.Duration) <-chan time.Time {
	return defaultWheel.After(d)
}

// Sleep .
func Sleep(d time.Duration) {
	defaultWheel.Sleep(d)
}

// AfterFunc .
func AfterFunc(d time.Duration, f func()) *Timer {
	return defaultWheel.AfterFunc(d, f)
}

// NewTimer .
func NewTimer(d time.Duration) *Timer {
	return defaultWheel.NewTimer(d)
}

// Reset .
func (t *Timer) Reset(d time.Duration) {
	t.r.w.resetTimer(t.r, d, 0)
}

// Stop .
func (t *Timer) Stop() {
	t.r.w.delTimer(t.r)
}
