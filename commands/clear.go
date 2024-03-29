package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type CommandClear struct{}

func (c *CommandClear) Name() string {
	return "clear"
}

func (c *CommandClear) Description() string {
	return "Очищает историю разговоров для текущего чата."
}

func (c *CommandClear) IsAdmin() bool {
	return false
}

func (c *CommandClear) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	chat.History = nil
	bot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров была очищена.")
}
