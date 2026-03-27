package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"strconv"
)

type CommandSummarize struct {
	Commands    *service.GPTCommandService
	ChatService *service.ChatService
	Notifier    *service.Notifier
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

func (c *CommandSummarize) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	messageCount := SummarizeDefaultMessageCount
	if len(ctx.CommandArgs) > 0 {
		messageCount, _ = strconv.Atoi(ctx.CommandArgs)
		if messageCount <= 0 {
			messageCount = SummarizeDefaultMessageCount
		}

		if messageCount > SummarizeMaxMessageCount {
			messageCount = SummarizeMaxMessageCount
		}
	}

	return summarizeText(c.Commands, c.ChatService, c.Notifier, chat, chat.Settings.SummarizePrompt, messageCount)
}
