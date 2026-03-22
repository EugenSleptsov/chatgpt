package commands

import (
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandSessionCurrent struct{}

func (c *CommandSessionCurrent) Name() string {
	return "current"
}

func (c *CommandSessionCurrent) Description() string {
	return "Показывает текущую активную сессию."
}

func (c *CommandSessionCurrent) IsAdmin() bool {
	return false
}

func (c *CommandSessionCurrent) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	s := chat.ActiveSession()
	tier := ai.FindTier(s.Model)
	modelLabel := s.Model
	if tier != nil {
		modelLabel = tier.Label
	}

	prompt := s.SystemPrompt
	if prompt == "" {
		prompt = "(не задан)"
	} else if len(prompt) > 100 {
		prompt = prompt[:100] + "..."
	}

	msg := fmt.Sprintf(
		"▶ Сессия #%d — %s\n\nМодель: %s\nСистемный промпт: %s\nСообщений: %d",
		s.ID, s.Topic, modelLabel, prompt, len(s.History),
	)
	return reply(msg)
}
