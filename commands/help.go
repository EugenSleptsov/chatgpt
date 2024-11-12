package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandHelp struct {
	TelegramBot     *telegram.Bot
	CommandRegistry CommandRegistry
}

func (c *CommandHelp) Name() string {
	return "help"
}

func (c *CommandHelp) Description() string {
	return "Показывает список доступных команд и их описание."
}

func (c *CommandHelp) IsAdmin() bool {
	return false
}

func (c *CommandHelp) Execute(update telegram.Update, chat *storage.Chat) {
	CommandList := c.CommandRegistry.GetCommands()

	message := "Список доступных команд и их описание:\n"
	var adminCommands []Command
	for _, command := range CommandList {
		if command.IsAdmin() {
			adminCommands = append(adminCommands, command)
			continue
		}

		message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
	}

	if c.TelegramBot.AdminId == update.Message.From.ID {
		message += "\nКоманды администратора:\n"
		for _, command := range adminCommands {
			message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
		}
	}

	c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, message)
}
