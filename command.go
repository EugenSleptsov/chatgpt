package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func commandRemoveUser(bot *telegram.Bot, update telegram.Update, chat *storage.Chat, config *Config) {
	chatID := chat.ChatID
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

func commandAddUser(bot *telegram.Bot, update telegram.Update, chat *storage.Chat, config *Config) {
	chatID := chat.ChatID
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

func commandReload(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	chatID := chat.ChatID
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)))
}

func commandTranslate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо перевести. Использование: /translate <text>")
	} else {
		prompt := update.Message.CommandArguments()
		translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks", prompt)
		systemPrompt := "You are a helpful assistant that translates."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, translationPrompt)
	}
}

func commandGrammar(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо скорректировать. Использование: /grammar <text>")
	} else {
		prompt := update.Message.CommandArguments()
		grammarPrompt := fmt.Sprintf("Correct the following text: \"%s\". Answer with corrected text only.", prompt)
		systemPrompt := "You are a helpful assistant that corrects grammar."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, grammarPrompt)
	}
}

func commandEnhance(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо улучшить. Использование: /enhance <text>")
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
/temperature <n> - Устанавливает температуру (креативность) для GPT. Допустимые значения: 0.0 - 1.2
/system <text> - Устанавливает системный промпт для GPT. Пример: "You are a helpful assistant that translates."
/summarize <n> - Генерирует краткое содержание последних <n> сообщений из истории разговоров для текущего чата. <n> по умолчанию равно 50.`
	bot.Reply(chat.ChatID, update.Message.MessageID, helpText)
}

func commandHistory(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	historyMessages := formatHistory(messagesFromHistory(chat.History))
	for _, message := range historyMessages {
		bot.Reply(chat.ChatID, update.Message.MessageID, message)
	}
}

func commandStart(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	bot.Reply(chat.ChatID, update.Message.MessageID, "Здравствуйте! Я чатбот-помощник, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help.")
}

func commandClear(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	chat.History = nil
	bot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров была очищена.")
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
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Удалено %d %s.", number, util.Pluralize(number, [3]string{"сообщение", "сообщения", "сообщений"})))
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров пуста.")
	}
}

func commandImagine(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat, config *Config) {
	now := time.Now()
	nextTime := chat.ImageGenNextTime
	if nextTime.After(now) && update.Message.From.ID != config.AdminId {
		nextTimeStr := nextTime.Format("15:04:05")
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Your next image generation will be available at %s.", nextTimeStr))
		return
	}

	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, по которому необходимо сгенерировать изображение. Использование: /imagine <text>")
	} else {
		chat.ImageGenNextTime = now.Add(time.Second * 900)
		gptImage(bot, chat.ChatID, gptClient, update.Message.CommandArguments(), config)
	}
}

func commandTemperature(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Текущая температура %.1f.", chat.Settings.Temperature))
	} else {
		temperature, err := strconv.ParseFloat(update.Message.CommandArguments(), 64)
		if err != nil || temperature < 0.0 || temperature > 1.2 {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Неверное значение температуры. Должно быть от 0.0 до 1.2.")
		} else {
			chat.Settings.Temperature = float32(temperature)
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Температура установлена на %.1f.", temperature))
		}
	}
}

func commandSystem(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		if chat.Settings.SystemPrompt == "" {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Системное сообщение не установлено.")
		} else {
			bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprint(chat.Settings.SystemPrompt))
		}
	} else {
		chat.Settings.SystemPrompt = update.Message.CommandArguments()
		if len(chat.Settings.SystemPrompt) > 1024 {
			chat.Settings.SystemPrompt = chat.Settings.SystemPrompt[:1024]
		}
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Системное сообщение установлено на: %s.", chat.Settings.SystemPrompt))
	}
}

func commandModel(bot *telegram.Bot, update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Текущая модель %s.", chat.Settings.Model))
	} else {
		model := update.Message.CommandArguments()
		switch model {
		case gpt.ModelGPT3, gpt.ModelGPT3Turbo:
			chat.Settings.Model = gpt.ModelGPT3Turbo
			bot.Reply(chat.ChatID, update.Message.MessageID, "Модель установлена на gpt-3.5-turbo.")
		case gpt.ModelGPT316k, gpt.ModelGPT316k2:
			chat.Settings.Model = gpt.ModelGPT316k
			bot.Reply(chat.ChatID, update.Message.MessageID, "Модель установлена на gpt-3.5-turbo-16k.")
		case gpt.ModelGPT4, gpt.ModelGPT4Preview:
			chat.Settings.Model = gpt.ModelGPT4Preview
			bot.Reply(chat.ChatID, update.Message.MessageID, "Модель установлена на gpt-4-1106-preview.")
		default:
			bot.Reply(chat.ChatID, update.Message.MessageID, "Неверное название модели.")
		}
	}
}

func commandSummarize(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	messageCount := 50
	if len(update.Message.CommandArguments()) > 0 {
		messageCount, _ = strconv.Atoi(update.Message.CommandArguments())
		if messageCount <= 0 {
			messageCount = 50
		}

		if messageCount > 100 {
			messageCount = 100
		}
	}

	bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Генерирую краткое содержание последних %d сообщений...", messageCount))
	// open log file
	lines, err := util.ReadLastLines(fmt.Sprintf("log/%d.log", chat.ChatID), messageCount)
	if err != nil {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Произошла ошибка")
		return
	}
	log.Printf("Lines: %v", lines)

	// cut lines to messageCount
	if len(lines) > messageCount {
		lines = lines[len(lines)-messageCount:]
	}
	if len(lines) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "История чата пуста")
		return
	}

	systemPrompt := "Ты - бот с острым языком и чувством юмора. Твоя задача - создать краткий смешной пересказ последних сообщений чата. Ты можешь добавлять свои комментарии в процессе пересказа, будто ты доктор Кокс или доктор Хаус. Не нужно передавать переписку дословно, также твое сообщение должно быть не более 300 слов (но не нужно насильно стремиться к этому числу)."
	chatLog := strings.Join(lines, "\n")
	gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, "Вот сообщения чата, которые ты должен обработать:\n\n"+chatLog)
}
