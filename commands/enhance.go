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

func (c *CommandEnhance) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if len(ctx.Msg.CommandArguments()) == 0 {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Пожалуйста укажите текст, который необходимо улучшить. Использование: /enhance <text>")
	} else {
		prompt := ctx.Msg.CommandArguments()
		enhancePrompt := fmt.Sprintf("Review and improve the following text: \"%s\". Answer with improved text only.", prompt)
		systemPrompt := "You are a helpful assistant that reviews text for grammar, style and things like that."
		gptText(c.Deps, chat, ctx.MessageID, systemPrompt, enhancePrompt)
	}
}
