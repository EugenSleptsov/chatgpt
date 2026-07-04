package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

// CommandSessionRename renames the ACTIVE session. It is the consumer of the
// "✏️ Имя" button in the session hub (list:rename → ForceReply → this command
// receives the new topic as args).
type CommandSessionRename struct{}

func (c *CommandSessionRename) Name() string {
	return "rename"
}

func (c *CommandSessionRename) Description() string {
	return "Переименовывает активную сессию (кнопка ✏️ в списке сессий)."
}

func (c *CommandSessionRename) IsAdmin() bool {
	return false
}

func (c *CommandSessionRename) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	topic := sanitizeTopic(ctx.CommandArgs)
	if topic == "" {
		return sessionListView(chat, sessionPageOf(chat, chat.ActiveSessionID))
	}
	chat.ActiveSession().Topic = topic
	return sessionListView(chat, sessionPageOf(chat, chat.ActiveSessionID))
}
