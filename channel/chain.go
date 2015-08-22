package channel

import (
	"sync"

	"github.com/gsdocker/gslogger"
)

// Context channel handle context
type Context interface {
	// ChannelWrite forward write data
	ChannelWrite(data []byte) error
	// ChannelRead backward read data
	ChannelRead(data []byte) error
}

// Handler gorpc channel filter handler
type Handler interface {
	// ChannelWrite write output data
	ChannelWrite(context Context, data []byte) error
	// ChannelRead read input data
	ChannelRead(context Context, data []byte) error
	// Close close handler
	Close()
}

// HandlerF handler create method
type HandlerF func() Handler

// _Chain channel handler chain
type _Chain struct {
	gslogger.Log           // mixin log APIs
	sync.Mutex             // mutex
	handlers     []Handler // handler chain
	readcursor   int       // process cursor
	writecursor  int       // write process cursor
}

func newChain(handlers []Handler) *_Chain {
	return &_Chain{
		Log:         gslogger.Get("channel"),
		handlers:    handlers,
		writecursor: len(handlers),
	}
}

func (chain *_Chain) ChannelWrite(data []byte) error {

	if chain.writecursor == 0 {
		// arrived tail handler
		return nil
	}

	chain.writecursor--

	handler := chain.handlers[chain.writecursor]

	err := handler.ChannelWrite(chain, data)

	chain.writecursor++

	return err
}

func (chain *_Chain) ChannelRead(data []byte) error {

	if chain.readcursor == len(chain.handlers) {
		// arrived tail handler
		return nil
	}

	handler := chain.handlers[chain.readcursor]

	chain.readcursor++

	err := handler.ChannelRead(chain, data)

	chain.readcursor--

	return err
}

// Handlers handler chain factory
type Handlers struct {
	creators []HandlerF
}

// NewHandlers .
func NewHandlers() *Handlers {
	return &Handlers{}
}

// Add .
func (handlers *Handlers) Add(f HandlerF) {
	handlers.creators = append(handlers.creators, f)
}

// Create create context
func (handlers *Handlers) Create() Context {
	var h []Handler

	for _, creator := range handlers.creators {
		h = append(h, creator())
	}

	return newChain(h)
}
