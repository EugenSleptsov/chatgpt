package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
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

func (c *CommandClear) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	service.ClearHistory(chat.ActiveSession())
	return reply("История разговоров была очищена.")
}
