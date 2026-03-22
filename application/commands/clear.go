package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

type CommandClear struct {
	History *service.HistoryService
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

func (c *CommandClear) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	c.History.Clear(chat.ActiveSession())
	return reply("История разговоров была очищена.")
}
