package commands

import (
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandSessionList struct{}

func (c *CommandSessionList) Name() string {
	return "list"
}

func (c *CommandSessionList) Description() string {
	return "Показывает список сессий (чатов)."
}

func (c *CommandSessionList) IsAdmin() bool {
	return false
}

func (c *CommandSessionList) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	msg := "📋 Сессии:\n\n"
	for _, s := range chat.Sessions {
		marker := "  "
		if s.ID == chat.ActiveSessionID {
			marker = "▶ "
		}
		tier := ai.FindTier(s.Model)
		modelLabel := s.Model
		if tier != nil {
			modelLabel = tier.Label
		}
		msg += fmt.Sprintf("%s#%d — %s [%s, %d сообщ.]\n", marker, s.ID, s.Topic, modelLabel, len(s.History))
	}
	msg += fmt.Sprintf("\nАктивная: #%d", chat.ActiveSessionID)
	return reply(msg)
}
