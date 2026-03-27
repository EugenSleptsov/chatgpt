package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

type CommandSessionNew struct{}

func (c *CommandSessionNew) Name() string {
	return "new"
}

func (c *CommandSessionNew) Description() string {
	return "Создаёт новую сессию и переключается на неё. Использование: /new <topic>"
}

func (c *CommandSessionNew) IsAdmin() bool {
	return false
}

func (c *CommandSessionNew) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	topic := strings.TrimSpace(ctx.CommandArgs)
	if topic == "" {
		topic = "untitled"
	}
	if len(topic) > 64 {
		topic = topic[:64]
	}

	s := chat.AddSession(topic)
	chat.ActiveSessionID = s.ID
	return reply(fmt.Sprintf("Создана и активирована сессия #%d — %s.", s.ID, s.Topic))
}
