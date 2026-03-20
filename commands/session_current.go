package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandSessionCurrent struct {
	*Deps
}

func (c *CommandSessionCurrent) Name() string {
	return "current"
}

func (c *CommandSessionCurrent) Description() string {
	return "Показывает текущую активную сессию."
}

func (c *CommandSessionCurrent) IsAdmin() bool {
	return false
}

func (c *CommandSessionCurrent) Execute(update telegram.Update, chat *storage.Chat) {
	s := chat.ActiveSession()
	tier := gpt.FindTier(s.Model)
	modelLabel := s.Model
	if tier != nil {
		modelLabel = tier.Label + " (" + tier.APIModel + ")"
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
	c.Bot.Reply(chat.ChatID, update.Message.MessageID, msg)
}
