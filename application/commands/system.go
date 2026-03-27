package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandSystem struct{}

func (c *CommandSystem) Name() string {
	return "system"
}

func (c *CommandSystem) Description() string {
	return "Устанавливает системный промпт для GPT. Пример: \"You are a helpful assistant that translates.\". Использование: /system <text>"
}

func (c *CommandSystem) IsAdmin() bool {
	return false
}

func (c *CommandSystem) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	session := chat.ActiveSession()
	if len(ctx.CommandArgs) == 0 {
		if session.SystemPrompt == "" {
			return reply("Системное сообщение не установлено.")
		}
		return reply(fmt.Sprint(session.SystemPrompt))
	}
	session.SystemPrompt = ctx.CommandArgs
	if len(session.SystemPrompt) > 1024 {
		session.SystemPrompt = session.SystemPrompt[:1024]
	}
	return reply(fmt.Sprintf("Системное сообщение установлено на: %s.", session.SystemPrompt))
}
