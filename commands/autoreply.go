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

func (c *CommandAutoReply) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	chat.Settings.GroupAutoReply = !chat.Settings.GroupAutoReply
	if chat.Settings.GroupAutoReply {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "✅ Авто-ответ включён. Бот будет самостоятельно вступать в разговор.")
	} else {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "❌ Авто-ответ выключен. Бот отвечает только при упоминании.")
	}
}
