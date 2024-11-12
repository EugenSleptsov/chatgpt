package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
)

type CommandHandler struct {
	TelegramClient *telegram.Bot
	CommandFactory commands.CommandFactory
}

func (c *CommandHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	command := update.Message.Command()

	cmd, err := c.CommandFactory.GetCommand(command)
	if err != nil {
		return err
	}

	if !cmd.IsAdmin() || update.Message.From.ID == c.TelegramClient.AdminId {
		cmd.Execute(update, chat)
	}

	return nil
}
