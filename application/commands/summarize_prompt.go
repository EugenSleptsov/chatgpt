package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandSummarizePrompt struct{}

func (c *CommandSummarizePrompt) Name() string {
	return "summarize_prompt"
}

func (c *CommandSummarizePrompt) Description() string {
	return "Устанавливает промпт для команды /summarize. Использование: /summarize_prompt <text>"
}

func (c *CommandSummarizePrompt) IsAdmin() bool {
	return false
}

func (c *CommandSummarizePrompt) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if len(ctx.CommandArgs) == 0 {
		return reply(fmt.Sprint("Текущий промпт для команды /summarize: ", chat.Settings.SummarizePrompt))
	}
	chat.Settings.SummarizePrompt = ctx.CommandArgs
	return reply("Промпт для команды /summarize установлен")
}
