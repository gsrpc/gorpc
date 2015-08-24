package gorpc

// Handler gorpc handler
type Handler interface {
	HandleStateChanged(router Router, state State) bool
	HandleRecieved(router Router, message *Message) (*Message, error)
	HandleSend(router Router, message *Message) (*Message, error)
	HandleClose(router Router)
	HandleError(router Router, err error) error
}

// HandlerF handler create method
type HandlerF func() Handler

// Handlers handler chain factory
type Handlers struct {
	creators []HandlerF
}

// NewHandlers .
func NewHandlers() *Handlers {
	return &Handlers{}
}

// Add .
func (handlers *Handlers) Add(f HandlerF) *Handlers {
	handlers.creators = append(handlers.creators, f)
	return handlers
}

// Create create context
func (handlers *Handlers) Create() []Handler {
	var h []Handler

	for _, creator := range handlers.creators {
		newhander := creator()

		if newhander != nil {
			h = append(h, newhander)
		}

	}

	return h
}
