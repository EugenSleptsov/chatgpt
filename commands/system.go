package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
)

type CommandSystem struct {
	*Deps
}

func (c *CommandSystem) Name() string {
	return "system"
}

func (c *CommandSystem) Description() string {
	return "Устанавливает системный промпт для GPT. Пример: \"You are a helpful assistant that translates.\". Использование: /system <text>"
}

func (c *CommandSystem) IsAdmin() bool {
	return false
}

func (c *CommandSystem) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	session := chat.ActiveSession()
	if len(ctx.Msg.CommandArguments()) == 0 {
		if session.SystemPrompt == "" {
			return reply("Системное сообщение не установлено.")
		}
		return reply(fmt.Sprint(session.SystemPrompt))
	}
	session.SystemPrompt = ctx.Msg.CommandArguments()
	if len(session.SystemPrompt) > 1024 {
		session.SystemPrompt = session.SystemPrompt[:1024]
	}
	return reply(fmt.Sprintf("Системное сообщение установлено на: %s.", session.SystemPrompt))
}
