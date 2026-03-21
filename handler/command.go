package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
)

type CommandHandler struct {
	Deps *commands.Deps
}

func (c *CommandHandler) Match(ctx *telegram.UpdateContext) bool {
	return ctx.IsCommand
}

// Handle executes the command directly. Commands are imperative —
// they manage their own responses, so we return nil.
func (c *CommandHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) *Request {
	cmd, err := c.Deps.Registry.GetCommand(ctx.Msg.Command())
	if err != nil {
		c.Deps.Notifier.Logf("Command not found: %v", err)
		return nil
	}

	if !cmd.IsAdmin() || c.Deps.Auth.IsAdmin(ctx.SenderID) {
		cmd.Execute(ctx, chat)
	}

	return nil
}
