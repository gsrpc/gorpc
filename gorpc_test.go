package gorpc

import (
	"testing"
	"time"

	"github.com/gsdocker/gslogger"
)

func TestPipeline(t *testing.T) {

	defer gslogger.Join()

	pipline, err := BuildPipeline().Handler(
		func() Handler {
			return LoggerHandler("one")
		},
	).Handler(
		func() Handler {
			return NewSink("test-sink", time.Second*5, 1024)
		},
	).Build("pipeline-test")

	if err != nil {
		t.Fatal(err)
	}

	message := NewMessage()

	message.Code = CodeHeartbeat

	err = pipline.ChannelWrite(message)

	if err != nil {
		t.Fatal(err)
	}

	eventHandler := HandleEvent(1)

	pipline.AddHandler(eventHandler)

	err = pipline.ChannelWrite(message)

	if err != nil {
		t.Fatal(err)
	}

	<-eventHandler.Write

	pipline.Close()

	err = pipline.ChannelWrite(message)

	if err != ErrClosed {
		t.Fatalf("test pipline closed error")
	}

	<-eventHandler.Close
}
