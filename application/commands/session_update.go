package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
	"strings"
)

type CommandSessionUpdate struct{}

func (c *CommandSessionUpdate) Name() string {
	return "update"
}

func (c *CommandSessionUpdate) Description() string {
	return "Переименовывает сессию. Использование: /update <id> <topic>"
}

func (c *CommandSessionUpdate) IsAdmin() bool {
	return false
}

func (c *CommandSessionUpdate) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	args := strings.SplitN(strings.TrimSpace(ctx.CommandArgs), " ", 2)
	if len(args) < 2 || args[1] == "" {
		return reply("Использование: /update <id> <topic>")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return reply("ID должен быть числом.")
	}

	s := chat.FindSession(id)
	if s == nil {
		return reply(fmt.Sprintf("Сессия #%d не найдена.", id))
	}

	topic := args[1]
	if len(topic) > 64 {
		topic = topic[:64]
	}

	old := s.Topic
	s.Topic = topic
	return reply(fmt.Sprintf("Сессия #%d переименована: %s → %s.", s.ID, old, s.Topic))
}
