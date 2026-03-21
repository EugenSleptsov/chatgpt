package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type CommandSessionNew struct {
	*Deps
}

func (c *CommandSessionNew) Name() string {
	return "new"
}

func (c *CommandSessionNew) Description() string {
	return "Создаёт новую сессию и переключается на неё. Использование: /new <topic>"
}

func (c *CommandSessionNew) IsAdmin() bool {
	return false
}

func (c *CommandSessionNew) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	topic := strings.TrimSpace(ctx.Msg.CommandArguments())
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
