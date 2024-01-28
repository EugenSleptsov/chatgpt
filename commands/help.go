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

func (c *CommandHelp) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	message := "Список доступных команд и их описание:\n"
	for _, command := range CommandList {
		message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
	}

	bot.Reply(update.Message.Chat.ID, update.Message.MessageID, message)
}
