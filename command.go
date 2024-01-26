package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	commands "GPTBot/commands"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func callCommand(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat, config *Config) {
	command := update.Message.Command()

	if commands.CommandList[command] != nil {
		commands.CommandList[command].Execute(bot, update, gptClient, chat)
		return
	}

	switch command {
	case "start":
		commandStart(bot, update, chat)
	case "clear":
		commandClear(bot, update, chat)
	case "history":
		commandHistory(bot, update, chat)
	case "rollback":
		commandRollback(bot, update, chat)
	case "help":
		commandHelp(bot, update, chat)
	case "translate":
		commandTranslate(bot, update, gptClient, chat)
	case "grammar":
		commandGrammar(bot, update, gptClient, chat)
	case "enhance":
		commandEnhance(bot, update, gptClient, chat)
	case "imagine":
		commandImagine(bot, update, gptClient, chat, config)
	case "temperature":
		commandTemperature(bot, update, chat)
	case "model":
		commandModel(bot, update, chat)
	case "system":
		commandSystem(bot, update, chat)
	case "summarize":
		commandSummarize(bot, update, gptClient, chat, config)
	default:
		if update.Message.From.ID != config.AdminId {
			// bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Неизвестная команда /%s", commands))
			break
		}

		switch command {
		case "reload":
			commandReload(bot, update, chat)
		case "adduser":
			commandAddUser(bot, update, chat, config)
		case "removeuser":
			commandRemoveUser(bot, update, chat, config)
		}
	}
}

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

func commandSummarize(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat, config *Config) {
	messageCount := 50
	if len(update.Message.CommandArguments()) > 0 {
		messageCount, _ = strconv.Atoi(update.Message.CommandArguments())
		if messageCount <= 0 {
			messageCount = 50
		}

		if messageCount > 500 {
			messageCount = 500
		}
	}

	// open log file
	lines, err := util.ReadLastLines(fmt.Sprintf("log/%d.log", chat.ChatID), messageCount)
	if err != nil {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Произошла ошибка")
		return
	}

	if len(lines) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "История чата пуста")
		return
	}

	bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Генерирую краткое содержание последних %d сообщений...", len(lines)))

	systemPrompt := config.SummarizePrompt
	if chat.ChatID > 0 { // private chats also can be summarized
		name := update.Message.From.FirstName + " " + update.Message.From.LastName
		systemPrompt = "Ты - бот с острым языком и чувством юмора. Твоя задача - создать краткий смешной пересказ сообщений твоего собеседника. Ты можешь добавлять свои комментарии в процессе пересказа, будто ты доктор Кокс или доктор Хаус. Не нужно передавать переписку дословно. На данный момент ты общаешься как раз со своим собеседником, которого зовут " + name + " и сообщение будет адресовано ему"
	}

	chatLog := strings.Join(lines, "\n")
	gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, "Вот сообщения чата, которые ты должен обработать:\n\n"+chatLog)
}

func gptText(bot *telegram.Bot, chat *storage.Chat, messageID int, gptClient *gpt.GPTClient, systemPrompt, userPrompt string) {
	responsePayload, err := gptClient.CallGPT35([]gpt.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, chat.Settings.Model, 0.6)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	response := "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	log.Printf("[%s] %s", "ChatGPT", response)
	bot.Reply(chat.ChatID, messageID, response)
}

func gptImage(bot *telegram.Bot, chatID int64, gptClient *gpt.GPTClient, prompt string, config *Config) {
	imageUrl, err := gptClient.GenerateImage(prompt, gpt.ImageSize1024)
	if err != nil {
		log.Printf("Error generating image: %v", err)
		return
	}

	enhancedCaption := prompt
	responsePayload, err := gptClient.CallGPT35([]gpt.Message{
		{Role: "system", Content: "You are an assistant that generates natural language description (prompt) for an artificial intelligence (AI) that generates images"},
		{Role: "user", Content: fmt.Sprintf("Please improve this prompt: \"%s\". Answer with improved prompt only. Keep prompt at most 200 characters long. Your prompt must be in one sentence.", prompt)},
	}, "gpt-3.5-turbo", 0.7)
	if err == nil {
		enhancedCaption = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	err = bot.SendImage(chatID, imageUrl, enhancedCaption)
	if err != nil {
		log.Printf("Error sending image: %v", err)
		return
	}

	log.Printf("[ChatGPT] sent image %s", imageUrl)
	if config.AdminId > 0 {
		if chatID != config.AdminId {
			bot.Message(fmt.Sprintf("Image with prompt \"%s\" sent to chat %d", prompt, chatID), config.AdminId, false)
		}
	}
}
