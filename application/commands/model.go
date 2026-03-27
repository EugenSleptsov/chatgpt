package commands

import (
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandModel struct{}

func (c *CommandModel) Name() string {
	return "model"
}

func (c *CommandModel) Description() string {
	return "Показывает или устанавливает модель. Использование: /model [ID]"
}

func (c *CommandModel) IsAdmin() bool {
	return false
}

func (c *CommandModel) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	session := chat.ActiveSession()
	args := ctx.CommandArgs

	if len(args) == 0 {
		current := ai.FindTier(session.Model)
		name := session.Model
		if current != nil {
			name = current.Label
		}
		return reply(fmt.Sprintf("Текущая модель: %s\n\nДоступные модели:\n%s", name, ai.TierList()))
	}

	tier := ai.FindTier(args)
	if tier == nil {
		return reply(fmt.Sprintf("Модель не найдена: %s\n\nДоступные модели:\n%s", args, ai.TierList()))
	}

	session.Model = tier.ID
	return reply(fmt.Sprintf("Модель установлена: %s (%s)", tier.Label, tier.Desc))
}
