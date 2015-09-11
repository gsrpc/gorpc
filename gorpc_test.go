package gorpc

import (
	"testing"
	"time"
)

func TestPipeline(t *testing.T) {

	eventHandler := HandleEvent(1)

	sink := NewSink("test-sink", time.Second*5, 1024, 10)

	pipline, err := BuildPipeline().Handler(
		"log",
		func() Handler {
			return LoggerHandler()
		},
	).Handler(
		"event",
		func() Handler {
			return eventHandler
		},
	).Handler(
		"sink",
		func() Handler {
			return sink
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

	<-eventHandler.Write

	err = pipline.ChannelWrite(message)

	if err != nil {
		t.Fatal(err)
	}

	<-eventHandler.Write

	sink.SendMessage(message)

	msg, err := pipline.ChannelRead()

	if err != nil {
		t.Fatal(err)
	}

	if msg != message {
		t.Fatal("test pipeline read err")
	}

	<-eventHandler.Read

	pipline.Close()

	err = pipline.ChannelWrite(message)

	if err != ErrClosed {
		t.Fatalf("test pipline closed error")
	}

	<-eventHandler.Close
}
