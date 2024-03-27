package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
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

func (c *CommandMarkdown) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if update.Message.CommandArguments() == "on" {
		chat.Settings.UseMarkdown = true
		bot.Reply(chat.ChatID, update.Message.MessageID, "Markdown включен")
	} else if update.Message.CommandArguments() == "off" {
		chat.Settings.UseMarkdown = false
		bot.Reply(chat.ChatID, update.Message.MessageID, "Markdown выключен")
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Текущее состояние Markdown: "+boolToString(chat.Settings.UseMarkdown))
		return
	}
}

func boolToString(b bool) string {
	if b {
		return "включен"
	}
	return "выключен"
}
