package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
)

type CommandHandler struct {
	Deps *commands.Deps
}

func (c *CommandHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	cmd, err := c.Deps.Registry.GetCommand(update.Message.Command())
	if err != nil {
		return err
	}

	if !cmd.IsAdmin() || c.Deps.Auth.IsAdmin(update.Message.From.ID) {
		cmd.Execute(update, chat)
	}

	return nil
}
