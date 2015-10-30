package trace

import "github.com/gsrpc/gorpc/snowflake"

// Consumer tracer consumer
type Consumer interface {
	// TraceFlag get trace flag
	TraceFlag() bool

	EvtRPC(rpc *EvtRPC)
}

type _TraceProbe struct {
	sf       *snowflake.SnowFlake
	consumer Consumer
}

func _NewTraceProbe(workID uint32, consumer Consumer) *_TraceProbe {
	return &_TraceProbe{
		sf:       snowflake.New(workID, 1024),
		consumer: consumer,
	}
}

// NewTrace create new trace id
func (tracer *_TraceProbe) NewTrace() uint64 {
	return <-tracer.sf.Gen
}

var _traceProbe *_TraceProbe

// Start open gsrpc trace
func Start(nodeID uint32, consumer Consumer) {
	_traceProbe = _NewTraceProbe(nodeID, consumer)
}

// NewTrace create a new trace id
func NewTrace() uint64 {
	if _traceProbe != nil {
		id := _traceProbe.NewTrace()
		return id
	}

	return 0
}

// Flag trace flag
func Flag() bool {
	if _traceProbe != nil {
		return _traceProbe.consumer.TraceFlag()
	}

	return false
}
