package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
)

type CommandModel struct{}

func (c *CommandModel) Name() string {
	return "model"
}

func (c *CommandModel) Description() string {
	return "Устанавливает модель для GPT."
}

func (c *CommandModel) IsAdmin() bool {
	return false
}

func (c *CommandModel) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Текущая модель %s.", chat.Settings.Model))
	} else {
		model := update.Message.CommandArguments()
		switch model {
		case gpt.ModelGPT3, gpt.ModelGPT3Turbo:
			chat.Settings.Model = gpt.ModelGPT3Turbo
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Модель установлена на %s", chat.Settings.Model))
		case gpt.ModelGPT316k, gpt.ModelGPT316k2:
			chat.Settings.Model = gpt.ModelGPT316k
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Модель установлена на %s", chat.Settings.Model))
		case gpt.ModelGPT4, gpt.ModelGPT4Preview:
			chat.Settings.Model = gpt.ModelGPT4Preview
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Модель установлена на %s", chat.Settings.Model))
		default:
			bot.Reply(chat.ChatID, update.Message.MessageID, "Неверное название модели.")
		}
	}
}
