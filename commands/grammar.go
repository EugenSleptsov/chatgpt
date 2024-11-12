package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandGrammar struct {
	TelegramBot *telegram.Bot
	GptClient   *gpt.GPTClient
}

func (c *CommandGrammar) Name() string {
	return "grammar"
}

func (c *CommandGrammar) Description() string {
	return "Исправляет грамматические ошибки в <text>. Использование: /grammar <text>"
}

func (c *CommandGrammar) IsAdmin() bool {
	return false
}

func (c *CommandGrammar) Execute(update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо скорректировать. Использование: /grammar <text>")
	} else {
		prompt := update.Message.CommandArguments()
		grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
		systemPrompt := "You are a helpful assistant that corrects grammar."
		gptText(c.TelegramBot, chat, update.Message.MessageID, c.GptClient, systemPrompt, grammarPrompt)
	}
}
