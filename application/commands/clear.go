package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"strings"
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

// Execute shows a confirmation keyboard (destructive action); the actual clear
// happens only on the "clear:yes" callback. Typed "/clear" lands on the confirm
// step too — tap to proceed.
func (c *CommandClear) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	switch strings.TrimSpace(ctx.CommandArgs) {
	case "yes":
		service.ClearHistory(chat.ActiveSession())
		return reply("История разговоров была очищена.")
	case "cancel":
		return reply("Отменено.")
	}
	return []sender.Response{{
		Text: "Очистить историю текущей сессии?",
		Buttons: [][]sender.Button{{
			{Text: "⚠️ Очистить", Data: "clear:yes"},
			{Text: "Отмена", Data: "clear:cancel"},
		}},
	}}
}
