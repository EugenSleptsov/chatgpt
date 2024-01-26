package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type CommandTest struct{}

func (c *CommandTest) Name() string {
	return "test"
}

func (c *CommandTest) Description() string {
	return "Test commands"
}

func (c *CommandTest) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	bot.Reply(update.Message.Chat.ID, update.Message.MessageID, "Test commands")
}
