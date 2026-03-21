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

func (c *CommandClear) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	chat.ActiveSession().History = nil
	c.Bot.Reply(chat.ChatID, ctx.MessageID, "История разговоров была очищена.")
}
