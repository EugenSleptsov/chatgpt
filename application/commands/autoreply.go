package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

type CommandAutoReply struct{}

func (c *CommandAutoReply) Name() string {
	return "autoreply"
}

func (c *CommandAutoReply) Description() string {
	return "Переключает режим авто-ответа: бот самостоятельно вступает в разговор группы."
}

func (c *CommandAutoReply) IsAdmin() bool {
	return true
}

func (c *CommandAutoReply) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	switch ctx.CommandArgs {
	case "on":
		chat.Settings.GroupAutoReply = true
	case "off":
		chat.Settings.GroupAutoReply = false
	}
	return boolView("autoreply", "Авто-ответ", chat.Settings.GroupAutoReply)
}
