package commands

import (
	"GPTBot/api/telegram"
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

func (c *CommandMarkdown) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if ctx.Msg.CommandArguments() == "on" {
		chat.Settings.UseMarkdown = true
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Markdown включен")
	} else if ctx.Msg.CommandArguments() == "off" {
		chat.Settings.UseMarkdown = false
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Markdown выключен")
	} else {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Текущее состояние Markdown: "+boolToString(chat.Settings.UseMarkdown))
		return
	}
}

func boolToString(b bool) string {
	if b {
		return "включен"
	}
	return "выключен"
}
