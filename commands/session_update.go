package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
	"strconv"
	"strings"
)

type CommandSessionUpdate struct {
	*Deps
}

func (c *CommandSessionUpdate) Name() string {
	return "update"
}

func (c *CommandSessionUpdate) Description() string {
	return "Переименовывает сессию. Использование: /update <id> <topic>"
}

func (c *CommandSessionUpdate) IsAdmin() bool {
	return false
}

func (c *CommandSessionUpdate) Execute(update telegram.Update, chat *storage.Chat) {
	args := strings.SplitN(strings.TrimSpace(update.Message.CommandArguments()), " ", 2)
	if len(args) < 2 || args[1] == "" {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Использование: /update <id> <topic>")
		return
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "ID должен быть числом.")
		return
	}

	s := chat.FindSession(id)
	if s == nil {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Сессия #%d не найдена.", id))
		return
	}

	topic := args[1]
	if len(topic) > 64 {
		topic = topic[:64]
	}

	old := s.Topic
	s.Topic = topic
	c.Bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Сессия #%d переименована: %s → %s.", s.ID, old, s.Topic))
}
