package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandEnhance struct {
	*Deps
}

func (c *CommandEnhance) Name() string {
	return "enhance"
}

func (c *CommandEnhance) Description() string {
	return "Улучшает <text> с помощью GPT. Использование: /enhance <text>"
}

func (c *CommandEnhance) IsAdmin() bool {
	return false
}

func (c *CommandEnhance) Execute(update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо улучшить. Использование: /enhance <text>")
	} else {
		prompt := update.Message.CommandArguments()
		enhancePrompt := fmt.Sprintf("Review and improve the following text: \"%s\". Answer with improved text only.", prompt)
		systemPrompt := "You are a helpful assistant that reviews text for grammar, style and things like that."
		c.gptText(chat, update.Message.MessageID, systemPrompt, enhancePrompt)
	}
}
