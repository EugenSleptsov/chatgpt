package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/util"
	"fmt"
	"log"
	"strconv"
	"time"
)

func commandRemoveUser(bot *telegram.Bot, update telegram.Update, chatID int64, config *Config) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to remove")
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		newList := make([]int64, 0)
		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User will be removed: %d", userId))
			} else {
				newList = append(newList, auth)
			}
		}

		config.AuthorizedUserIds = newList
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, "Command successfully ended")
	}
}

func commandAddUser(bot *telegram.Bot, update telegram.Update, chatID int64, config *Config) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to add")
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		for _, auth := range config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User already added: %d", userId))
				return
			}
		}

		config.AuthorizedUserIds = append(config.AuthorizedUserIds, userId)
		err = updateConfig("bot.conf", config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User successfully added: %d", userId))
	}
}

func commandReload(bot *telegram.Bot, update telegram.Update, chatID int64) {
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)))
}

func commandTranslate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chatID int64) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a text to translate. Usage: /translate <text>")
	} else {
		prompt := update.Message.CommandArguments()
		translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks", prompt)
		systemPrompt := "You are a helpful assistant that translates."
		gptText(bot, chatID, update.Message.MessageID, gptClient, systemPrompt, translationPrompt)
	}
}

func commandGrammar(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chatID int64) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a text to correct. Usage: /grammar <text>")
	} else {
		prompt := update.Message.CommandArguments()
		grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
		systemPrompt := "You are a helpful assistant that corrects grammar."
		gptText(bot, chatID, update.Message.MessageID, gptClient, systemPrompt, grammarPrompt)
	}
}

func commandEnhance(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chatID int64) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a text to enhance. Usage: /enhance <text>")
	} else {
		prompt := update.Message.CommandArguments()
		enhancePrompt := fmt.Sprintf("Review and improve the following text: \"%s\". Answer with improved text only.", prompt)
		systemPrompt := "You are a helpful assistant that reviews text for grammar, style and things like that."
		gptText(bot, chatID, update.Message.MessageID, gptClient, systemPrompt, enhancePrompt)
	}
}

func commandHelp(bot *telegram.Bot, update telegram.Update, chatID int64) {
	helpText := `Список доступных команд и их описание:
/help - Показывает список доступных команд и их описание.
/start - Отправляет приветственное сообщение, описывающее цель бота.
/history - Показывает всю сохраненную на данный момент историю разговоров в красивом форматировании.
/clear - Очищает историю разговоров для текущего чата.
/rollback <n> - Удаляет последние <n> сообщений из истории разговоров для текущего чата.
/translate <text> - Переводит <text> на любом языке на английский язык
/grammar <text> - Исправляет грамматические ошибки в <text>
/enhance <text> - Улучшает <text> с помощью GPT
/imagine <text> - Генерирует изображение по описанию <text> размера 512x512`
	bot.Reply(chatID, update.Message.MessageID, helpText)
}

func commandHistory(bot *telegram.Bot, update telegram.Update, chatID int64) {
	historyMessages := formatHistory(messagesFromHistory(chatHistory[chatID]))
	for _, message := range historyMessages {
		bot.Reply(chatID, update.Message.MessageID, message)
	}
}

func commandStart(bot *telegram.Bot, update telegram.Update, chatID int64) {
	bot.Reply(chatID, update.Message.MessageID, "Здравствуйте! Я помощник GPT-3.5 Turbo, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help")
}

func commandClear(bot *telegram.Bot, update telegram.Update, chatID int64) {
	chatHistory[chatID] = nil
	bot.Reply(chatID, update.Message.MessageID, "История разговоров была очищена.")
}

func commandRollback(bot *telegram.Bot, update telegram.Update, chatID int64) {
	number := 1
	if len(update.Message.CommandArguments()) > 0 {
		var err error
		number, err = strconv.Atoi(update.Message.CommandArguments())
		if err != nil || number < 1 {
			number = 1
		}
	}

	if number > len(chatHistory[chatID]) {
		number = len(chatHistory[chatID])
	}

	if len(chatHistory[chatID]) > 0 {
		chatHistory[chatID] = chatHistory[chatID][:len(chatHistory[chatID])-number]
		bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Удалено %d %s.", number, util.Pluralize(number, [3]string{"сообщение", "сообщения", "сообщений"})))
	} else {
		bot.Reply(chatID, update.Message.MessageID, "История разговоров пуста.")
	}
}

func commandImagine(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chatID int64, config *Config) {
	now := time.Now()
	nextTime, exists := imageGenNextTime[chatID]
	if exists && nextTime.After(now) && chatID != config.AdminId && update.Message.From.ID != config.AdminId {
		nextTimeStr := nextTime.Format("15:04:05")
		bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Your next image generation will be available at %s.", nextTimeStr))
		return
	}

	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a text to generate an image. Usage: /image <text>")
	} else {
		imageGenNextTime[chatID] = now.Add(time.Second * 900)
		gptImage(bot, chatID, gptClient, update.Message.CommandArguments(), config)
	}
}
