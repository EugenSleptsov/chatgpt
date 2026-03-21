package commands

import (
	"GPTBot/api/telegram"
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

func (c *CommandSystem) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	session := chat.ActiveSession()
	if len(ctx.Msg.CommandArguments()) == 0 {
		if session.SystemPrompt == "" {
			c.Bot.Reply(chat.ChatID, ctx.MessageID, "Системное сообщение не установлено.")
		} else {
			c.Bot.Reply(chat.ChatID, ctx.MessageID, fmt.Sprint(session.SystemPrompt))
		}
	} else {
		session.SystemPrompt = ctx.Msg.CommandArguments()
		if len(session.SystemPrompt) > 1024 {
			session.SystemPrompt = session.SystemPrompt[:1024]
		}
		c.Bot.Reply(chat.ChatID, ctx.MessageID, fmt.Sprintf("Системное сообщение установлено на: %s.", session.SystemPrompt))
	}
}
