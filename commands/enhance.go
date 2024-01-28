package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandEnhance struct{}

func (c *CommandEnhance) Name() string {
	return "enhance"
}

func (c *CommandEnhance) Description() string {
	return "Show enhance"
}

func (c *CommandEnhance) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо улучшить. Использование: /enhance <text>")
	} else {
		prompt := update.Message.CommandArguments()
		enhancePrompt := fmt.Sprintf("Review and improve the following text: \"%s\". Answer with improved text only.", prompt)
		systemPrompt := "You are a helpful assistant that reviews text for grammar, style and things like that."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, enhancePrompt)
	}
}
