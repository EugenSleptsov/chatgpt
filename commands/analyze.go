package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"strconv"
	"strings"
)

type CommandAnalyze struct {
	*Deps
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

func (c *CommandAnalyze) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if len(ctx.Msg.CommandArguments()) == 0 {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Пожалуйста укажите количество сообщений (опционально) и промпт для обработки. Использование: /analyze <count> <prompt>")
		return
	}

	var systemPrompt string
	arguments := strings.Split(ctx.Msg.CommandArguments(), " ")
	messageCount, err := strconv.Atoi(arguments[0])
	if err != nil {
		messageCount = AnalyzeDefaultMessageCount
		systemPrompt = ctx.Msg.CommandArguments()
	} else {
		if len(arguments) < 2 {
			c.Bot.Reply(chat.ChatID, ctx.MessageID, "Пожалуйста укажите промпт для обработки. Использование: /analyze <count> <prompt>")
			return
		}

		systemPrompt = strings.Join(arguments[1:], " ")
		if messageCount <= 0 {
			messageCount = AnalyzeDefaultMessageCount
		}

		if messageCount > AnalyzeMaxMessageCount {
			messageCount = AnalyzeMaxMessageCount
		}
	}

	summarizeText(c.Deps, chat, ctx.MessageID, systemPrompt, messageCount)
}
