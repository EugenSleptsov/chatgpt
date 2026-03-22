package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
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

func (c *CommandAutoReply) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	chat.Settings.GroupAutoReply = !chat.Settings.GroupAutoReply
	if chat.Settings.GroupAutoReply {
		return reply("✅ Авто-ответ включён. Бот будет самостоятельно вступать в разговор.")
	}
	return reply("❌ Авто-ответ выключен. Бот отвечает только при упоминании.")
}
