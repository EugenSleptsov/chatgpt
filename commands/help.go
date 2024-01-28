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
	return "Returns list of available commands"
}

func (c *CommandHelp) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	message := "Available commands:\n"
	for _, command := range CommandList {
		if command.Name() == c.Name() {
			continue
		}

		message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
	}

	bot.Reply(update.Message.Chat.ID, update.Message.MessageID, message)
}
