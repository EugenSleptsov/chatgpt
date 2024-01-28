package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"strconv"
)

type CommandSummarize struct{}

func (c *CommandSummarize) Name() string {
	return "summarize"
}

func (c *CommandSummarize) Description() string {
	return "Генерирует краткое содержание последних <n> сообщений из истории разговоров для текущего чата. <n> по умолчанию равно 50. Максимальное значение <n> равно 500. Использование: /summarize <n>"
}

func (c *CommandSummarize) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	messageCount := 50
	if len(update.Message.CommandArguments()) > 0 {
		messageCount, _ = strconv.Atoi(update.Message.CommandArguments())
		if messageCount <= 0 {
			messageCount = 50
		}

		if messageCount > 500 {
			messageCount = 500
		}
	}

	summarizeText(bot, chat, update.Message.MessageID, gptClient, chat.Settings.SummarizePrompt, messageCount)
}
