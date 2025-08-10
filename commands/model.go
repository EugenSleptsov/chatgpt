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
		// Показываем outer-модель
		c.TelegramBot.Reply(
			chat.ChatID,
			update.Message.MessageID,
			fmt.Sprintf("Текущая модель: %s", gpt.MapModelName(chat.Settings.Model)),
		)
		return
	}

	model := update.Message.CommandArguments()
	switch model {
	case gpt.ModelGPT3:
		chat.Settings.Model = gpt.ModelGPT3
	case gpt.ModelGPT4OmniMini:
		chat.Settings.Model = gpt.ModelGPT4OmniMini
	case gpt.ModelGPT4:
		chat.Settings.Model = gpt.ModelGPT4Omni
	case gpt.ModelGPT5:
		chat.Settings.Model = gpt.ModelGPT5
	case gpt.ModelGPT5Nano:
		chat.Settings.Model = gpt.ModelGPT5Nano
	default:
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Неверное название модели.")
		return
	}

	// Сообщаем outer-модель, чтобы было понятно, что реально используется
	c.TelegramBot.Reply(
		chat.ChatID,
		update.Message.MessageID,
		fmt.Sprintf("Модель установлена на %s", gpt.MapModelName(chat.Settings.Model)),
	)
}
