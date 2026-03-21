package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
)

type CommandModel struct {
	*Deps
}

func (c *CommandModel) Name() string {
	return "model"
}

func (c *CommandModel) Description() string {
	return "Показывает или устанавливает модель. Использование: /model [ID]"
}

func (c *CommandModel) IsAdmin() bool {
	return false
}

func (c *CommandModel) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	session := chat.ActiveSession()
	args := ctx.Msg.CommandArguments()

	if len(args) == 0 {
		current := gpt.FindTier(session.Model)
		name := session.Model
		if current != nil {
			name = current.Label + " (" + current.APIModel + ")"
		}
		return reply(fmt.Sprintf("Текущая модель: %s\n\nДоступные модели:\n%s", name, gpt.TierList()))
	}

	tier := gpt.FindTier(args)
	if tier == nil {
		return reply(fmt.Sprintf("Модель не найдена: %s\n\nДоступные модели:\n%s", args, gpt.TierList()))
	}

	session.Model = tier.ID
	return reply(fmt.Sprintf("Модель установлена: %s (%s)", tier.Label, tier.Desc))
}
