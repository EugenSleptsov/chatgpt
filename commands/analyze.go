package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"strconv"
	"strings"
)

type CommandAnalyze struct{}

func (c *CommandAnalyze) Name() string {
	return "analyze"
}

func (c *CommandAnalyze) Description() string {
	return "Обработка логов чата с использованием переданного промпта"
}

func (c *CommandAnalyze) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите количество сообщений (опционально) и промпт для обработки. Использование: /analyze <count> <text>")
		return
	}

	var systemPrompt string
	arguments := strings.Split(update.Message.CommandArguments(), " ")
	messageCount, err := strconv.Atoi(arguments[0])
	if err != nil {
		messageCount = 50
		systemPrompt = update.Message.CommandArguments()
	} else {
		if len(arguments) < 2 {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите промпт для обработки. Использование: /analyze <count> <text>")
			return
		}

		systemPrompt = strings.Join(arguments[1:], " ")
		if messageCount <= 0 {
			messageCount = 50
		}
	}

	summarizeText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, messageCount)
}
