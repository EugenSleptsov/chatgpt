package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandGrammar struct{}

func (c *CommandGrammar) Name() string {
	return "grammar"
}

func (c *CommandGrammar) Description() string {
	return "Исправляет грамматические ошибки в <text>. Использование: /grammar <text>"
}

func (c *CommandGrammar) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо скорректировать. Использование: /grammar <text>")
	} else {
		prompt := update.Message.CommandArguments()
		grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
		systemPrompt := "You are a helpful assistant that corrects grammar."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, grammarPrompt)
	}
}
