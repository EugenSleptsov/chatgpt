package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
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

func (c *CommandEnhance) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	if len(ctx.Msg.CommandArguments()) == 0 {
		return reply("Пожалуйста укажите текст, который необходимо улучшить. Использование: /enhance <text>")
	}
	prompt := ctx.Msg.CommandArguments()
	enhancePrompt := fmt.Sprintf("Review and improve the following text: \"%s\". Answer with improved text only.", prompt)
	systemPrompt := "You are a helpful assistant that reviews text for grammar, style and things like that."
	return gptText(c.Deps, chat, systemPrompt, enhancePrompt)
}
