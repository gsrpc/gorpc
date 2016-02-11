package snowflake

import "time"

// .
const (
	workerIDBits = 10              // worker id
	maxWorkerID  = -1 ^ (-1 << 10) // worker id mask
	sequenceBits = 12              // sequence
	maxSequence  = -1 ^ (-1 << 12) //sequence mask
	nano         = 1000 * 1000
)

var (
	since = time.Date(2012, 1, 0, 0, 0, 0, 0, time.UTC).UnixNano() / nano
)

// SnowFlake .
type SnowFlake struct {
	workerID uint32
	sequence uint32
	closed   chan bool
	Gen      chan uint64
}

// New create new snowflake id generator
func New(workID uint32, cached uint32) *SnowFlake {
	sf := &SnowFlake{
		workerID: workID,
		closed:   make(chan bool),
		Gen:      make(chan uint64, cached),
	}

	go func() {

		for {
			id := (timestamp() << (workerIDBits + sequenceBits)) |
				(uint64(sf.workerID) << sequenceBits) |
				(uint64(sf.sequence))

			sf.sequence++

			select {
			case sf.Gen <- id:
			case <-sf.closed:
				return
			}
		}
	}()

	return sf
}

func timestamp() uint64 {
	return uint64(time.Now().UnixNano()/nano - since)
}
