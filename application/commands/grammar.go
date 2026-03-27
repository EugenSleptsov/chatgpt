package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandGrammar struct {
	Commands *service.GPTCommandService
	Notifier *service.Notifier
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

func (c *CommandGrammar) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if len(ctx.CommandArgs) == 0 {
		return reply("Пожалуйста укажите текст, который необходимо скорректировать. Использование: /grammar <text>")
	}
	prompt := ctx.CommandArgs
	grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
	systemPrompt := "You are a helpful assistant that corrects grammar."
	return gptText(c.Commands, c.Notifier, chat, systemPrompt, grammarPrompt)
}
