package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandSummarizePrompt struct {
	*Deps
}

func (c *CommandSummarizePrompt) Name() string {
	return "summarize_prompt"
}

func (c *CommandSummarizePrompt) Description() string {
	return "Устанавливает промпт для команды /summarize. Использование: /summarize_prompt <text>"
}

func (c *CommandSummarizePrompt) IsAdmin() bool {
	return false
}

func (c *CommandSummarizePrompt) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if len(ctx.Msg.CommandArguments()) == 0 {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, fmt.Sprint("Текущий промпт для команды /summarize: ", chat.Settings.SummarizePrompt))
	} else {
		chat.Settings.SummarizePrompt = ctx.Msg.CommandArguments()
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Промпт для команды /summarize установлен")
	}
}
