package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandSessionList struct {
	*Deps
}

func (c *CommandSessionList) Name() string {
	return "list"
}

func (c *CommandSessionList) Description() string {
	return "Показывает список сессий (чатов)."
}

func (c *CommandSessionList) IsAdmin() bool {
	return false
}

func (c *CommandSessionList) Execute(update telegram.Update, chat *storage.Chat) {
	msg := "📋 Сессии:\n\n"
	for _, s := range chat.Sessions {
		marker := "  "
		if s.ID == chat.ActiveSessionID {
			marker = "▶ "
		}
		tier := gpt.FindTier(s.Model)
		modelLabel := s.Model
		if tier != nil {
			modelLabel = tier.Label
		}
		msg += fmt.Sprintf("%s#%d — %s [%s, %d сообщ.]\n", marker, s.ID, s.Topic, modelLabel, len(s.History))
	}
	msg += fmt.Sprintf("\nАктивная: #%d", chat.ActiveSessionID)
	c.Bot.Reply(chat.ChatID, update.Message.MessageID, msg)
}
