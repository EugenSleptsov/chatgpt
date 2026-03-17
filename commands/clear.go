package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type CommandClear struct {
	*Deps
}

func (c *CommandClear) Name() string {
	return "clear"
}

func (c *CommandClear) Description() string {
	return "Очищает историю разговоров для текущего чата."
}

func (c *CommandClear) IsAdmin() bool {
	return false
}

func (c *CommandClear) Execute(update telegram.Update, chat *storage.Chat) {
	chat.History = nil
	c.Bot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров была очищена.")
}
