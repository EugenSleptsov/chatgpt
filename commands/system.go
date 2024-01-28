package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandSystem struct{}

func (c *CommandSystem) Name() string {
	return "system"
}

func (c *CommandSystem) Description() string {
	return "Show system information"
}

func (c *CommandSystem) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		if chat.Settings.SystemPrompt == "" {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Системное сообщение не установлено.")
		} else {
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprint(chat.Settings.SystemPrompt))
		}
	} else {
		chat.Settings.SystemPrompt = update.Message.CommandArguments()
		if len(chat.Settings.SystemPrompt) > 1024 {
			chat.Settings.SystemPrompt = chat.Settings.SystemPrompt[:1024]
		}
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Системное сообщение установлено на: %s.", chat.Settings.SystemPrompt))
	}
}
