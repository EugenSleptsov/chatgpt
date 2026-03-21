package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

// CommandBranch executes a recognised bot command and returns responses.
type CommandBranch interface {
	Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Response
}

// Dispatcher is the processing layer that routes a normalised Request
// to the appropriate branch:
//   - command requests  -> CommandBranch
//   - everything else   -> conversational Pipeline
type Dispatcher struct {
	Commands       CommandBranch
	Conversational *Pipeline
}

// NewDispatcher creates a Dispatcher wired to both branches.
func NewDispatcher(commands CommandBranch, pipeline *Pipeline) *Dispatcher {
	return &Dispatcher{Commands: commands, Conversational: pipeline}
}

// Process routes the request and returns the collected responses.
func (d *Dispatcher) Process(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Response {
	if req.CommandName != "" {
		return d.Commands.Execute(ctx, chat, req)
	}
	return d.Conversational.Process(ctx, chat, req)
}
