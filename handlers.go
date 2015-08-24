package gorpc

import "github.com/gsdocker/gslogger"

type _LoggerHandler struct {
	gslogger.Log
}

// LoggerHandler open log handler
func LoggerHandler(name string) Handler {
	return &_LoggerHandler{
		Log: gslogger.Get(name),
	}
}

func (handler *_LoggerHandler) OpenHandler(context Context) error {
	handler.D("open handlers")
	return nil
}

func (handler *_LoggerHandler) CloseHandler(context Context) {
	handler.D("close handlers")
}

func (handler *_LoggerHandler) HandleWrite(context Context, message *Message) (*Message, error) {

	handler.D("write message(%s)", message.Code)

	return message, nil
}

func (handler *_LoggerHandler) HandleRead(context Context, message *Message) (*Message, error) {

	handler.D("read message(%s)", message.Code)

	return message, nil
}

func (handler *_LoggerHandler) HandleError(context Context, err error) error {

	handler.D("handle err :%s", err)

	return err
}

// EventHandler .
type EventHandler struct {
	Open  chan bool
	Close chan bool
	Write chan *Message
	Read  chan *Message
	Error chan error
}

// HandleEvent open EventHandler
func HandleEvent(cachedsize int) *EventHandler {
	return &EventHandler{
		Open:  make(chan bool, cachedsize),
		Close: make(chan bool, cachedsize),
		Write: make(chan *Message, cachedsize),
		Read:  make(chan *Message, cachedsize),
		Error: make(chan error, cachedsize),
	}
}

// OpenHandler .
func (handler *EventHandler) OpenHandler(context Context) error {
	handler.Open <- true
	return nil
}

// CloseHandler .
func (handler *EventHandler) CloseHandler(context Context) {

	handler.Close <- true
}

// HandleWrite .
func (handler *EventHandler) HandleWrite(context Context, message *Message) (*Message, error) {
	clone := *message

	handler.Write <- &clone

	return message, nil
}

// HandleRead .
func (handler *EventHandler) HandleRead(context Context, message *Message) (*Message, error) {

	clone := *message

	handler.Read <- &clone

	return message, nil
}

// HandleError .
func (handler *EventHandler) HandleError(context Context, err error) error {

	handler.Error <- err

	return err
}
