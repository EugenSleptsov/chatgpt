package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strconv"
	"time"
)

func commandRemoveUser(bot *telegram.Bot, update telegram.Update, chat *storage.Chat, config *Config) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to remove", false)
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()), false)
			return
		}

		newList := make([]int64, 0)
		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User will be removed: %d", userId), false)
			} else {
				newList = append(newList, auth)
			}
		}

		config.AuthorizedUserIds = newList
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, "Command successfully ended", false)
	}
}

func commandAddUser(bot *telegram.Bot, update telegram.Update, chat *storage.Chat, config *Config) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to add", false)
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()), false)
			return
		}

		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User already added: %d", userId), false)
				return
			}
		}

		config.AuthorizedUserIds = append(config.AuthorizedUserIds, userId)
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User successfully added: %d", userId), false)
	}
}

func commandReload(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	chatID := chat.ChatID
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)), false)
}

func commandTranslate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Please provide a text to translate. Usage: /translate <text>", false)
	} else {
		prompt := update.Message.CommandArguments()
		translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks", prompt)
		systemPrompt := "You are a helpful assistant that translates."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, translationPrompt)
	}
}

func commandGrammar(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Please provide a text to correct. Usage: /grammar <text>", false)
	} else {
		prompt := update.Message.CommandArguments()
		grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
		systemPrompt := "You are a helpful assistant that corrects grammar."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, grammarPrompt)
	}
}

func commandEnhance(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Please provide a text to enhance. Usage: /enhance <text>", false)
	} else {
		prompt := update.Message.CommandArguments()
		enhancePrompt := fmt.Sprintf("Review and improve the following text: \"%s\". Answer with improved text only.", prompt)
		systemPrompt := "You are a helpful assistant that reviews text for grammar, style and things like that."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, enhancePrompt)
	}
}

func commandHelp(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	helpText := `Список доступных команд и их описание:
/help - Показывает список доступных команд и их описание.
/start - Отправляет приветственное сообщение, описывающее цель бота.
/history - Показывает всю сохраненную на данный момент историю разговоров в красивом форматировании.
/clear - Очищает историю разговоров для текущего чата.
/rollback <n> - Удаляет последние <n> сообщений из истории разговоров для текущего чата.
/translate <text> - Переводит <text> на любом языке на английский язык
/grammar <text> - Исправляет грамматические ошибки в <text>
/enhance <text> - Улучшает <text> с помощью GPT
/imagine <text> - Генерирует изображение по описанию <text> размера 512x512
/temperature <n> - Устанавливает температуру (креативность) для GPT. Допустимые значения: 0.0 - 1.2`
	bot.Reply(chat.ChatID, update.Message.MessageID, helpText, false)
}

func commandHistory(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	historyMessages := formatHistory(messagesFromHistory(chat.History))
	for _, message := range historyMessages {
		bot.Reply(chat.ChatID, update.Message.MessageID, message, false)
	}
}

func commandStart(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	bot.Reply(chat.ChatID, update.Message.MessageID, "Здравствуйте! Я помощник GPT-3.5 Turbo, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help. ```Hehe```", true)
}

func commandClear(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	chat.History = nil
	bot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров была очищена.", false)
}

func commandRollback(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	number := 1
	if len(update.Message.CommandArguments()) > 0 {
		var err error
		number, err = strconv.Atoi(update.Message.CommandArguments())
		if err != nil || number < 1 {
			number = 1
		}
	}

	if number > len(chat.History) {
		number = len(chat.History)
	}

	if len(chat.History) > 0 {
		chat.History = chat.History[:len(chat.History)-number]
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Удалено %d %s.", number, util.Pluralize(number, [3]string{"сообщение", "сообщения", "сообщений"})), false)
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров пуста.", false)
	}
}

func commandImagine(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat, config *Config) {
	now := time.Now()
	nextTime := chat.ImageGenNextTime
	if nextTime.After(now) && update.Message.From.ID != config.AdminId {
		nextTimeStr := nextTime.Format("15:04:05")
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Your next image generation will be available at %s.", nextTimeStr), false)
		return
	}

	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Please provide a text to generate an image. Usage: /image <text>", false)
	} else {
		chat.ImageGenNextTime = now.Add(time.Second * 900)
		gptImage(bot, chat.ChatID, gptClient, update.Message.CommandArguments(), config)
	}
}

func commandTemperature(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Текущая температура %.1f.", chat.Settings.Temperature), false)
	} else {
		temperature, err := strconv.ParseFloat(update.Message.CommandArguments(), 64)
		if err != nil || temperature < 0.0 || temperature > 1.2 {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Неверное значение температуры. Должно быть от 0.0 до 1.2.", false)
		} else {
			chat.Settings.Temperature = float32(temperature)
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Температура установлена на %.1f.", temperature), false)
		}
	}
}
