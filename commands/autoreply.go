package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type CommandAutoReply struct {
	*Deps
}

func (c *CommandAutoReply) Name() string {
	return "autoreply"
}

func (c *CommandAutoReply) Description() string {
	return "Переключает режим авто-ответа: бот самостоятельно вступает в разговор группы."
}

func (c *CommandAutoReply) IsAdmin() bool {
	return true
}

func (c *CommandAutoReply) Execute(update telegram.Update, chat *storage.Chat) {
	chat.Settings.GroupAutoReply = !chat.Settings.GroupAutoReply
	if chat.Settings.GroupAutoReply {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "✅ Авто-ответ включён. Бот будет самостоятельно вступать в разговор.")
	} else {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "❌ Авто-ответ выключен. Бот отвечает только при упоминании.")
	}
}
