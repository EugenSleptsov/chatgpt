package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strings"
	"time"
)

func start(bot *telegram.Bot, gptClient *gpt.GPTClient, botStorage storage.Storage, config *Config) {
	// buffer up to 100 update messages
	updateChan := make(chan telegram.Update, 100)

	// create a pool of worker goroutines
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		go worker(updateChan, bot, gptClient, botStorage, config)
	}

	for update := range bot.GetUpdateChannel(config.TimeoutValue) {
		// Ignore any non-Message Updates
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		chat, ok := botStorage.Get(chatID)
		if !ok {
			chat = &storage.Chat{
				ChatID: update.Message.Chat.ID,
				Settings: storage.ChatSettings{
					Temperature:     0.8,
					Model:           "gpt-3.5-turbo",
					MaxMessages:     config.MaxMessages,
					UseMarkdown:     false,
					SystemPrompt:    "You are a helpful ChatGPT bot based on OpenAI GPT Language model. You are a helpful assistant that always tries to help and answer with relevant information as possible.",
					SummarizePrompt: config.SummarizePrompt,
				},
				History:          make([]*storage.ConversationEntry, 0),
				ImageGenNextTime: time.Now(),
			}
			_ = botStorage.Set(chatID, chat)
		}

		if !update.Message.IsCommand() {
			// putting history to log file
			// every newline is a new message
			var lines []string
			name := update.Message.From.FirstName + " " + update.Message.From.LastName
			for _, v := range strings.Split(update.Message.Text, "\n") {
				if v != "" {
					lines = append(lines, v)
				}
			}

			// для групповых чатов указываем имя пользователя
			if chat.ChatID < 0 {
				for i := range lines {
					lines[i] = fmt.Sprintf("%s: %s", name, lines[i])
				}
			}

			// saving lines to log file
			util.AddLines(fmt.Sprintf("log/%d.log", chat.ChatID), lines)
		}

		// If no authorized users are provided, make the bot public
		if len(config.AuthorizedUserIds) > 0 {
			if !util.IsIdInList(update.Message.From.ID, config.AuthorizedUserIds) {
				if update.Message.Chat.Type == "private" {
					bot.Reply(chatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
					attemptMessage := fmt.Sprintf("Unauthorized access attempt by user %d: %s %s (@%s)", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName)
					log.Print(attemptMessage)

					// Notify the admin
					if config.AdminId > 0 {
						bot.Message(attemptMessage, config.AdminId, false)
					}
				}
				continue
			}
		}

		// Send the Update to the worker goroutines via the channel
		updateChan <- update
	}
}

// worker function that processes updates
func worker(updateChan <-chan telegram.Update, bot *telegram.Bot, gptClient *gpt.GPTClient, botStorage storage.Storage, config *Config) {
	for update := range updateChan {
		processUpdate(bot, update, gptClient, botStorage, config)
		botStorage.Save()
	}
}

func processUpdate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, botStorage storage.Storage, config *Config) {
	chatID := update.Message.Chat.ID
	chat, _ := botStorage.Get(chatID)

	if update.Message.Voice != nil {
		response, err := processAudio(bot, gptClient, update.Message.Voice.FileID)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}

		bot.Reply(chatID, update.Message.MessageID, response)

		// check if message is forwarded, then we finish here
		if update.Message.ForwardFrom != nil {
			// send admin message that transcribe was done
			if config.AdminId > 0 {
				bot.Message(fmt.Sprintf("Transcribe for user %s %s (@%s)", update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName), config.AdminId, false)
			}
			return
		}
		update.Message.Text = response
	}

	// Check for commands
	if update.Message.IsCommand() {
		callCommand(bot, update, gptClient, chat, config)
	} else {
		callReply(bot, update, gptClient, chat, config)
	}
}

func callReply(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat, config *Config) {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	if chat.ChatID < 0 { // group chat
		isReplyToBot := update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == bot.Username
		if !strings.Contains(update.Message.Text, "@"+bot.Username) && !isReplyToBot {
			return
		}

		if strings.Contains(update.Message.Text, "@"+bot.Username) {
			update.Message.Text = strings.Replace(update.Message.Text, "@"+bot.Username, "", -1)
		}
	}

	// Maintain conversation history
	userMessage := storage.Message{Role: "user", Content: update.Message.Text}
	historyEntry := &storage.ConversationEntry{Prompt: userMessage, Response: storage.Message{}}

	chat.History = append(chat.History, historyEntry)
	if len(chat.History) > chat.Settings.MaxMessages {
		excessMessages := len(chat.History) - chat.Settings.MaxMessages
		chat.History = chat.History[excessMessages:]
	}

	var messages []gpt.Message
	if chat.Settings.SystemPrompt != "" {
		messages = append(messages, gpt.Message{Role: "system", Content: chat.Settings.SystemPrompt})
	}
	for _, entry := range chat.History {
		messages = append(messages, gpt.Message{Role: entry.Prompt.Role, Content: entry.Prompt.Content})
		if entry.Response != (storage.Message{}) {
			messages = append(messages, gpt.Message{Role: entry.Response.Role, Content: entry.Response.Content})
		}
	}

	response := "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	responsePayload, err := gptClient.CallGPT35(messages, chat.Settings.Model, chat.Settings.Temperature)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	}

	// Add the assistant's response to the conversation history
	historyEntry.Response = storage.Message{Role: "assistant", Content: response}

	log.Printf("[%s] %s", "ChatGPT", response)
	if chat.Settings.UseMarkdown {
		bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response)
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, response)
	}

	notifyAdmin(bot, config, update, response)
}

func notifyAdmin(bot *telegram.Bot, config *Config, update telegram.Update, response string) {
	if config.AdminId == 0 {
		return
	}

	if update.Message.From.ID == config.AdminId {
		return
	}

	if util.IsIdInList(update.Message.From.ID, config.IgnoreReportIds) {
		return
	}

	bot.Message(fmt.Sprintf("[User: %s %s (%s, ID: %d)] %s\n[ChatGPT] %s\n", update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName, update.Message.From.ID, update.Message.Text, response), config.AdminId, false)
}
