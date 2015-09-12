package gorpc

import (
	"runtime"
	"testing"
	"time"
)

var eventLoop = NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

type _NullMessageChannel struct {
}

func (null *_NullMessageChannel) SendMessage(message *Message) error {
	return nil
}

func (null *_NullMessageChannel) Close() {

}

func TestPipeline(t *testing.T) {

	pipeline, err := BuildPipeline(eventLoop).Handler("profile", ProfileHandler).Build("test", &_NullMessageChannel{})

	if err != nil {
		t.Fatal(err)
	}

	err = pipeline.Active()

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10000; i++ {
		pipeline.Received(nil)
	}

	for i := 0; i < 10000; i++ {
		pipeline.Send(nil)
	}

	pipeline.Inactive()

	pipeline.Close()

	println(PrintProfile())
}
