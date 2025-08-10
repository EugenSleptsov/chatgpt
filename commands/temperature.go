package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
	"strconv"
)

type CommandTemperature struct {
	TelegramBot *telegram.Bot
}

func (c *CommandTemperature) Name() string {
	return "temperature"
}

func (c *CommandTemperature) Description() string {
	return "Устанавливает температуру (креативность) для GPT. Допустимые значения: 0.0 - 1.2."
}

func (c *CommandTemperature) IsAdmin() bool {
	return true
}

func (c *CommandTemperature) Execute(update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Текущая температура %.1f.", chat.Settings.Temperature))
	} else {
		temperature, err := strconv.ParseFloat(update.Message.CommandArguments(), 64)
		if err != nil || temperature < 0.0 || temperature > 1.2 {
			c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Неверное значение температуры. Должно быть от 0.0 до 1.2.")
		} else {
			chat.Settings.Temperature = float32(temperature)
			c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Температура установлена на %.1f.", temperature))
		}
	}
}
