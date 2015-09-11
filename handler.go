package gorpc

// Context channel handler context
type Context interface {
	// current handler name
	Name() string
	// Pipeline current channel pipeline
	Pipeline() Pipeline
	// OnActive
	OnActive()
	// OnMessageReceived fire MessageReceived event on next pipline handler
	OnMessageReceived(message *Message)
	// OnMessageSending fire MessageSending event on prev pipline handler
	OnMessageSending(message *Message)
	// Send create new send pipeline message
	Send(message *Message) error
}

// Handler the gorpc channel pipeline handlers
type Handler interface {
	// Register when handler had been add into one pipeline,
	// system call this function notify handler
	Register(context Context) error
	// Unregister sysm call this function when handler had been removed from
	// pipeline,you can get this pipeline object by Context#Pipeline function
	Unregister(context Context)
	// Active system call this function when pipline state trans to active
	Active(context Context) error
	// Inactive system call this function when pipeline state trans to inactive
	Inactive(context Context)
	// MessageReceived
	MessageReceived(context Context, message *Message) error
	// MessageSending
	MessageSending(context Context, message *Message) error
	// Panic handle async pipline method error
	Panic(context Context, err error)
}

// SharedHandler this handler will been shared with more than one piplines
type SharedHandler interface {
	// Lock lock this handler
	Lock()
	// Unlock unlock this handler
	Unlock()
}

// HandlerF handler factory
type HandlerF func() Handler

type _Context struct {
	name     string        // context bound handler name
	handler  Handler       // context bound handler
	shared   SharedHandler // shared handler
	next     *_Context     // pipeline next handler
	prev     *_Context     // pipeline prev handler
	pipeline *_Pipeline    // pipeline which handler belongs to
}

func newContext(name string, handler Handler, pipeline *_Pipeline, prev *_Context) (*_Context, error) {
	context := &_Context{
		name:     name,
		handler:  handler,
		pipeline: pipeline,
		prev:     prev,
	}

	context.shared, _ = handler.(SharedHandler)

	context.lock()
	defer context.unlock()

	err := context.handler.Register(context)

	if err != nil {
		return nil, err
	}

	if context.prev != nil {
		context.prev.next = context
	}

	return context, nil

}

func (context *_Context) lock() {
	if context.shared != nil {
		context.shared.Lock()
	}
}

func (context *_Context) unlock() {
	if context.shared != nil {
		context.shared.Unlock()
	}
}

func (context *_Context) Name() string {
	return context.name
}

func (context *_Context) Pipeline() Pipeline {
	return context.pipeline
}

func (context *_Context) OnActive() {

}

func (context *_Context) OnMessageReceived(message *Message) {

}

func (context *_Context) OnMessageSending(message *Message) {

}

func (context *_Context) Send(message *Message) error {
	return nil
}
