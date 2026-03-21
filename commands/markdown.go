package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
)

type CommandMarkdown struct {
	*Deps
}

func (c *CommandMarkdown) Name() string {
	return "markdown"
}

func (c *CommandMarkdown) Description() string {
	return "Использование Markdown в сообщениях. Использование: /markdown <on|off>"
}

func (c *CommandMarkdown) IsAdmin() bool {
	return false
}

func (c *CommandMarkdown) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	if ctx.Msg.CommandArguments() == "on" {
		chat.Settings.UseMarkdown = true
		return reply("Markdown включен")
	} else if ctx.Msg.CommandArguments() == "off" {
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
