package trace

import (
	"fmt"
	"time"

	"github.com/gsrpc/gorpc"
)

// RPC .
func RPC(trace uint64, id uint32, parent uint32) *EvtRPC {

	evt := NewEvtRPC()

	evt.Trace = trace
	evt.ID = id
	evt.Prev = parent

	return evt
}

// Start start record rpc
func (evt *EvtRPC) Start() {
	now := time.Now()
	evt.StartTime.Second = uint64(now.Unix())
	evt.StartTime.Nano = uint64(now.UnixNano())
}

// End end record rpc
func (evt *EvtRPC) End() {
	now := time.Now()
	evt.EndTime.Second = uint64(now.Unix())
	evt.EndTime.Nano = uint64(now.UnixNano())

	if _traceProbe != nil {
		_traceProbe.consumer.EvtRPC(evt)
	}
}

// Set append new attributes
func (evt *EvtRPC) Set(key []byte, val []byte) {
	evt.Attributes = append(evt.Attributes, &gorpc.KV{Key: key, Value: val})
}

func (evt *EvtRPC) String() string {
	return fmt.Sprintf("TRACE:%d-%d call from %d", evt.Trace, evt.ID, evt.Prev)
}
