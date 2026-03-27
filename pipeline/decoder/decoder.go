package decoder

import (
	"GPTBot/pipeline"
	"GPTBot/pipeline/executor"
)

// Decoder dispatches updates to the first matching executor,
// in the order they were registered.
type Decoder struct {
	executors []executor.Executor
}

// NewDecoder creates an empty Decoder. Register executors via Register().
func NewDecoder() *Decoder {
	return &Decoder{}
}

// Register adds an executor to the decoder.
func (d *Decoder) Register(e executor.Executor) {
	d.executors = append(d.executors, e)
}

// RegisterAll adds multiple executors in order.
func (d *Decoder) RegisterAll(executors []executor.Executor) {
	d.executors = append(d.executors, executors...)
}

// Decode returns the first executor whose Match returns true, or nil.
func (d *Decoder) Decode(ctx *pipeline.RequestContext) executor.Executor {
	for _, e := range d.executors {
		if e.Match(ctx) {
			return e
		}
	}
	return nil
}
