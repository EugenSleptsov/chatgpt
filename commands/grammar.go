package commands

import (
	"GPTBot/api/telegram"
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

func (c *CommandGrammar) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if len(ctx.Msg.CommandArguments()) == 0 {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Пожалуйста укажите текст, который необходимо скорректировать. Использование: /grammar <text>")
	} else {
		prompt := ctx.Msg.CommandArguments()
		grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
		systemPrompt := "You are a helpful assistant that corrects grammar."
		gptText(c.Deps, chat, ctx.MessageID, systemPrompt, grammarPrompt)
	}
}
