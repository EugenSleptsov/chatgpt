package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"strings"
)

// CommandSessionNew creates a new session and switches to it. It is the
// consumer of the "➕ Сессия" button in the session hub (list:new → ForceReply
// → this command receives the topic as args).
type CommandSessionNew struct{}

func (c *CommandSessionNew) Name() string {
	return "new"
}

func (c *CommandSessionNew) Description() string {
	return "Создаёт новую сессию (кнопка ➕ в списке сессий)."
}

func (c *CommandSessionNew) IsAdmin() bool {
	return false
}

func (c *CommandSessionNew) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	topic := sanitizeTopic(ctx.CommandArgs)
	if topic == "" {
		topic = "untitled"
	}

	s := chat.AddSession(topic)
	chat.ActiveSessionID = s.ID
	return sessionListView(chat, sessionPageOf(chat, s.ID))
}

// sanitizeTopic trims and caps a user-supplied session topic.
func sanitizeTopic(raw string) string {
	topic := strings.TrimSpace(raw)
	if len(topic) > 64 {
		topic = topic[:64]
	}
	return topic
}
