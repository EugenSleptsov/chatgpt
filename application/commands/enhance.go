package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandEnhance struct {
	Commands *service.GPTCommandService
	Notifier *service.Notifier
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

func (c *CommandEnhance) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if len(ctx.CommandArgs) == 0 {
		return reply("Пожалуйста укажите текст, который необходимо улучшить. Использование: /enhance <text>")
	}
	prompt := ctx.CommandArgs
	enhancePrompt := fmt.Sprintf("Review and improve the following text: \"%s\". Answer with improved text only.", prompt)
	systemPrompt := "You are a helpful assistant that reviews text for grammar, style and things like that."
	return gptText(c.Commands, c.Notifier, chat, systemPrompt, enhancePrompt)
}
