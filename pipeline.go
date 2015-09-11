package gorpc

// Pipeline Channel handlers pipeline
type Pipeline interface {
	// Name pipeline name
	Name() string
	// Close close pipeline
	Close()
	// Active trans pipeline state to active state
	Active() error
	// Inactive trans pipeline state to inactive state
	Inactive()
}

// PipelineBuilder pipeline builder
type PipelineBuilder struct {
	handlers []HandlerF // handler factories
	names    []string   // handler names
	executor EventLoop  // event loop
}

// BuildPipeline creaet new pipeline builder
func BuildPipeline(executor EventLoop) *PipelineBuilder {
	return &PipelineBuilder{
		executor: executor,
	}
}

// Handler append new handler builder
func (builder *PipelineBuilder) Handler(name string, handlerF HandlerF) *PipelineBuilder {

	builder.handlers = append(builder.handlers, handlerF)

	builder.names = append(builder.names, name)

	return builder
}

type _Pipeline struct {
	name     string    // pipeline name
	header   *_Context // pipeline head handler
	executor EventLoop // event loop
}

// Build create new Pipeline
func (builder *PipelineBuilder) Build(name string) (Pipeline, error) {

	pipeline := &_Pipeline{
		name:     name,
		executor: builder.executor,
	}

	var err error

	for i, f := range builder.handlers {
		pipeline.header, err = newContext(builder.names[i], f(), pipeline, pipeline.header)

		if err != nil {
			return nil, err
		}
	}

	return pipeline, nil
}

func (pipeline *_Pipeline) Name() string {
	return pipeline.name
}

func (pipeline *_Pipeline) Close() {

}

func (pipeline *_Pipeline) Active() error {
	return nil
}

func (pipeline *_Pipeline) Inactive() {

}

func (pipeline *_Pipeline) onActive(context *_Context) {
	pipeline.executor.Execute(func() {

	})
}

func (pipeline *_Pipeline) onMessageReceived(message *Message) {

}

func (pipeline *_Pipeline) onMessageSending(message *Message) {

}

func (context *_Context) send(message *Message) error {
	return nil
}
