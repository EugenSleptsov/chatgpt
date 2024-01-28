package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandTranslate struct{}

func (c *CommandTranslate) Name() string {
	return "translate"
}

func (c *CommandTranslate) Description() string {
	return "Translate message"
}

func (c *CommandTranslate) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо перевести. Использование: /translate <text>")
	} else {
		prompt := update.Message.CommandArguments()
		translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks", prompt)
		systemPrompt := "You are a helpful assistant that translates."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, translationPrompt)
	}
}
