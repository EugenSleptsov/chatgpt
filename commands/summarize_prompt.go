package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandSummarizePrompt struct {
	TelegramBot *telegram.Bot
}

func (c *CommandSummarizePrompt) Name() string {
	return "summarize_prompt"
}

func (c *CommandSummarizePrompt) Description() string {
	return "Устанавливает промпт для команды /summarize. Использование: /summarize_prompt <text>"
}

func (c *CommandSummarizePrompt) IsAdmin() bool {
	return false
}

func (c *CommandSummarizePrompt) Execute(update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprint("Текущий промпт для команды /summarize: ", chat.Settings.SummarizePrompt))
	} else {
		chat.Settings.SummarizePrompt = update.Message.CommandArguments()
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Промпт для команды /summarize установлен")
	}
}
