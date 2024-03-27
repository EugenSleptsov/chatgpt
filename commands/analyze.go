package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"strconv"
	"strings"
)

type CommandAnalyze struct{}

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

func (c *CommandAnalyze) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите количество сообщений (опционально) и промпт для обработки. Использование: /analyze <count> <prompt>")
		return
	}

	var systemPrompt string
	arguments := strings.Split(update.Message.CommandArguments(), " ")
	messageCount, err := strconv.Atoi(arguments[0])
	if err != nil {
		messageCount = AnalyzeDefaultMessageCount
		systemPrompt = update.Message.CommandArguments()
	} else {
		if len(arguments) < 2 {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите промпт для обработки. Использование: /analyze <count> <prompt>")
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

	summarizeText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, messageCount)
}
