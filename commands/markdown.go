package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type CommandMarkdown struct {
	TelegramBot *telegram.Bot
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

func (c *CommandMarkdown) Execute(update telegram.Update, chat *storage.Chat) {
	if update.Message.CommandArguments() == "on" {
		chat.Settings.UseMarkdown = true
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Markdown включен")
	} else if update.Message.CommandArguments() == "off" {
		chat.Settings.UseMarkdown = false
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Markdown выключен")
	} else {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Текущее состояние Markdown: "+boolToString(chat.Settings.UseMarkdown))
		return
	}
}

func boolToString(b bool) string {
	if b {
		return "включен"
	}
	return "выключен"
}
