package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandModel struct {
	TelegramBot *telegram.Bot
}

func (c *CommandModel) Name() string {
	return "model"
}

func (c *CommandModel) Description() string {
	return "Устанавливает модель для GPT."
}

func (c *CommandModel) IsAdmin() bool {
	return false
}

func (c *CommandModel) Execute(update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Текущая модель %s.", chat.Settings.Model))
	} else {
		model := update.Message.CommandArguments()
		switch model {
		case gpt.ModelGPT3:
		case gpt.ModelGPT4OmniMini:
			chat.Settings.Model = gpt.ModelGPT4OmniMini
			c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Модель установлена на %s", chat.Settings.Model))
		case gpt.ModelGPT4:
			chat.Settings.Model = gpt.ModelGPT4Omni
			c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Модель установлена на %s", chat.Settings.Model))
		default:
			c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Неверное название модели.")
		}
	}
}
