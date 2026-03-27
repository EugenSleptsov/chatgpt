package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"strconv"
	"strings"
)

type CommandAnalyze struct {
	Commands    *service.GPTCommandService
	ChatService *service.ChatService
	Notifier    *service.Notifier
}

const AnalyzeDefaultMessageCount = 50
const AnalyzeMaxMessageCount = 500

func (c *CommandAnalyze) Name() string {
	return "analyze"
}

func (c *CommandAnalyze) Description() string {
	return "Генерирует краткое содержание последних <n> сообщений из истории разговоров для текущего чата с использованием переданного промпта. Использование: /analyze <count> <prompt>"
}

func (c *CommandAnalyze) IsAdmin() bool {
	return false
}

func (c *CommandAnalyze) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if len(ctx.CommandArgs) == 0 {
		return reply("Пожалуйста укажите количество сообщений (опционально) и промпт для обработки. Использование: /analyze <count> <prompt>")
	}

	var systemPrompt string
	arguments := strings.Split(ctx.CommandArgs, " ")
	messageCount, err := strconv.Atoi(arguments[0])
	if err != nil {
		messageCount = AnalyzeDefaultMessageCount
		systemPrompt = ctx.CommandArgs
	} else {
		if len(arguments) < 2 {
			return reply("Пожалуйста укажите промпт для обработки. Использование: /analyze <count> <prompt>")
		}

		systemPrompt = strings.Join(arguments[1:], " ")
		if messageCount <= 0 {
			messageCount = AnalyzeDefaultMessageCount
		}

		if messageCount > AnalyzeMaxMessageCount {
			messageCount = AnalyzeMaxMessageCount
		}
	}

	return summarizeText(c.Commands, c.ChatService, c.Notifier, chat, systemPrompt, messageCount)
}
