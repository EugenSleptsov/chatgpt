package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

type CommandMarkdown struct{}

func (c *CommandMarkdown) Name() string {
	return "markdown"
}

func (c *CommandMarkdown) Description() string {
	return "Использование Markdown в сообщениях. Использование: /markdown <on|off>"
}

func (c *CommandMarkdown) IsAdmin() bool {
	return false
}

func (c *CommandMarkdown) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if ctx.CommandArgs == "on" {
		chat.Settings.UseMarkdown = true
		return reply("Markdown включен")
	} else if ctx.CommandArgs == "off" {
		chat.Settings.UseMarkdown = false
		return reply("Markdown выключен")
	}
	return reply("Текущее состояние Markdown: " + boolToString(chat.Settings.UseMarkdown))
}

func boolToString(b bool) string {
	if b {
		return "включен"
	}
	return "выключен"
}
