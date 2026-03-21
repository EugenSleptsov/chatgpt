package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
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

func (c *CommandClear) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	chat.ActiveSession().History = nil
	return reply("История разговоров была очищена.")
}
