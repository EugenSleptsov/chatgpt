package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandHelp struct{}

func (c *CommandHelp) Name() string {
	return "help"
}

func (c *CommandHelp) Description() string {
	return "Показывает список доступных команд и их описание."
}

func (c *CommandHelp) IsAdmin() bool {
	return false
}

func (c *CommandHelp) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	message := "Список доступных команд и их описание:\n"
	var adminCommands []Command
	for _, command := range CommandList {
		if command.IsAdmin() {
			adminCommands = append(adminCommands, command)
			continue
		}

		message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
	}

	if bot.AdminId == update.Message.From.ID {
		message += "\nКоманды администратора:\n"
		for _, command := range adminCommands {
			message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
		}
	}

	bot.Reply(update.Message.Chat.ID, update.Message.MessageID, message)
}
