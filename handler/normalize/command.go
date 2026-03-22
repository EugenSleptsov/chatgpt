package normalize

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
)

// CommandHandler normalises a command update into a Request.
// Actual command lookup and execution is done by the command branch
// in the processing layer.
type CommandHandler struct{}

func (c *CommandHandler) Match(ctx *telegram.UpdateContext) bool {
	return ctx.IsCommand
}

// Handle produces a Request that carries the command name and arguments.
// The processing layer (Dispatcher) routes it to the command branch.
func (c *CommandHandler) Handle(ctx *telegram.UpdateContext, _ *storage.Chat) *handler.Request {
	return &handler.Request{
		CommandName: ctx.Msg.Command(),
		CommandArgs: ctx.Msg.CommandArguments(),
	}
}
