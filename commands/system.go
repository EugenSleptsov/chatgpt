package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandSystem struct {
	TelegramBot *telegram.Bot
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

func (c *CommandSystem) Execute(update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		if chat.Settings.SystemPrompt == "" {
			c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Системное сообщение не установлено.")
		} else {
			c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprint(chat.Settings.SystemPrompt))
		}
	} else {
		chat.Settings.SystemPrompt = update.Message.CommandArguments()
		if len(chat.Settings.SystemPrompt) > 1024 {
			chat.Settings.SystemPrompt = chat.Settings.SystemPrompt[:1024]
		}
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Системное сообщение установлено на: %s.", chat.Settings.SystemPrompt))
	}
}
