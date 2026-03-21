package commands

import (
	"GPTBot/api/telegram"
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

func (c *CommandSessionNew) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	topic := strings.TrimSpace(ctx.Msg.CommandArguments())
	if topic == "" {
		topic = "untitled"
	}
	if len(topic) > 64 {
		topic = topic[:64]
	}

	s := chat.AddSession(topic)
	chat.ActiveSessionID = s.ID
	c.Bot.Reply(chat.ChatID, ctx.MessageID, fmt.Sprintf("Создана и активирована сессия #%d — %s.", s.ID, s.Topic))
}
