package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandSummarizePrompt struct{}

func (c *CommandSummarizePrompt) Name() string {
	return "summarize_prompt"
}

func (c *CommandSummarizePrompt) Description() string {
	return "Show summarize_prompt"
}

func (c *CommandSummarizePrompt) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprint("Текущий промпт для команды /summarize: ", chat.Settings.SummarizePrompt))
	} else {
		chat.Settings.SummarizePrompt = update.Message.CommandArguments()
		bot.Reply(chat.ChatID, update.Message.MessageID, "Промпт для команды /summarize установлен")
	}
}
