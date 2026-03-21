package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

// Pipeline resolves intents from a normalized Request and executes them
// in order, collecting all responses.
type Pipeline struct {
	resolver  *IntentResolver
	executors map[IntentType]IntentExecutor
}

// NewPipeline creates an empty Pipeline with the given resolver.
// Register executors via RegisterExecutor().
func NewPipeline(resolver *IntentResolver) *Pipeline {
	return &Pipeline{
		resolver:  resolver,
		executors: make(map[IntentType]IntentExecutor),
	}
}

// RegisterExecutor binds an executor to an intent type.
func (p *Pipeline) RegisterExecutor(t IntentType, exec IntentExecutor) {
	p.executors[t] = exec
}

// Process resolves intents and runs the matching executors sequentially.
func (p *Pipeline) Process(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Response {
	intents := p.resolver.Resolve(ctx, chat, req)

	var responses []Response
	for _, intent := range intents {
		if exec, ok := p.executors[intent.Type]; ok {
			responses = append(responses, exec.Execute(ctx, chat, req)...)
		}
	}
	return responses
}
