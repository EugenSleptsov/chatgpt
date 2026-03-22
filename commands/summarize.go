package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"strconv"
)

type CommandSummarize struct {
	*Deps
}

const SummarizeDefaultMessageCount = 50
const SummarizeMaxMessageCount = 500

func (c *CommandSummarize) Name() string {
	return "summarize"
}

func (c *CommandSummarize) Description() string {
	return "Генерирует краткое содержание последних <n> сообщений из истории разговоров для текущего чата. <n> по умолчанию равно 50. Максимальное значение <n> равно 500. Использование: /summarize <n>"
}

func (c *CommandSummarize) IsAdmin() bool {
	return false
}

func (c *CommandSummarize) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	messageCount := SummarizeDefaultMessageCount
	if len(ctx.Msg.CommandArguments()) > 0 {
		messageCount, _ = strconv.Atoi(ctx.Msg.CommandArguments())
		if messageCount <= 0 {
			messageCount = SummarizeDefaultMessageCount
		}

		if messageCount > SummarizeMaxMessageCount {
			messageCount = SummarizeMaxMessageCount
		}
	}

	return summarizeText(c.Deps, chat, chat.Settings.SummarizePrompt, messageCount)
}
