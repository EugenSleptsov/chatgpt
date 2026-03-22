package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
)

type CommandGrammar struct {
	*Deps
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

func (c *CommandGrammar) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	if len(ctx.Msg.CommandArguments()) == 0 {
		return reply("Пожалуйста укажите текст, который необходимо скорректировать. Использование: /grammar <text>")
	}
	prompt := ctx.Msg.CommandArguments()
	grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
	systemPrompt := "You are a helpful assistant that corrects grammar."
	return gptText(c.Deps, chat, systemPrompt, grammarPrompt)
}
