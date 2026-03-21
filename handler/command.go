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

func (c *CommandHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) error {
	cmd, err := c.Deps.Registry.GetCommand(ctx.Msg.Command())
	if err != nil {
		return err
	}

	if !cmd.IsAdmin() || c.Deps.Auth.IsAdmin(ctx.SenderID) {
		cmd.Execute(ctx, chat)
	}

	return nil
}
