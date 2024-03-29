package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strings"
	"time"
)

func start(bot *telegram.Bot, gptClient *gpt.GPTClient, botStorage storage.Storage) {
	// buffer up to 100 update messages
	updateChan := make(chan telegram.Update, 100)

	// create a pool of worker goroutines
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		go worker(updateChan, bot, gptClient, botStorage)
	}

	for update := range bot.GetUpdateChannel(bot.Config.TimeoutValue) {
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
					MaxMessages:     bot.Config.MaxMessages,
					UseMarkdown:     false,
					SystemPrompt:    "You are a helpful ChatGPT bot based on OpenAI GPT Language model. You are a helpful assistant that always tries to help and answer with relevant information as possible.",
					SummarizePrompt: bot.Config.SummarizePrompt,
					Token:           bot.Config.GPTToken,
				},
				History:          make([]*storage.ConversationEntry, 0),
				ImageGenNextTime: time.Now(),
				Title:            telegram.GetChatTitle(update),
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
			_ = util.AddLines(fmt.Sprintf("log/%d.log", chat.ChatID), lines)
		}

		// If no authorized users are provided, make the bot public
		if len(bot.Config.AuthorizedUserIds) > 0 {
			if !util.IsIdInList(update.Message.From.ID, bot.Config.AuthorizedUserIds) {
				if update.Message.Chat.Type == "private" {
					bot.Reply(chatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
					bot.Log(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, update.Message.Text))
				}
				continue
			}
		}

		// Send the Update to the worker goroutines via the channel
		updateChan <- update
	}
}

// worker function that processes updates
func worker(updateChan <-chan telegram.Update, bot *telegram.Bot, gptClient *gpt.GPTClient, botStorage storage.Storage) {
	for update := range updateChan {
		processUpdate(bot, update, gptClient, botStorage)
		botStorage.Save()
	}
}

func processUpdate(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, botStorage storage.Storage) {
	chatID := update.Message.Chat.ID
	chat, _ := botStorage.Get(chatID)
	chat.Title = telegram.GetChatTitle(update)

	if update.Message.Voice != nil {
		response, err := processAudio(bot, gptClient, update.Message.Voice.FileID)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}

		bot.Reply(chatID, update.Message.MessageID, response)

		// check if message is forwarded, then we finish here
		if update.Message.ForwardFrom != nil {
			bot.Log(fmt.Sprintf("[%s] %s", telegram.GetChatTitle(update), "Transcribe was done"))
			return
		}
		update.Message.Text = response
	}

	if len(update.Message.Photo) > 0 {
		callImageReply(bot, update, gptClient, chat)
		return
	}

	// Check for commands
	if update.Message.IsCommand() {
		callCommand(bot, update, gptClient, chat)
	} else {
		callReply(bot, update, gptClient, chat)
	}
}

func callImageReply(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	image := update.Message.Photo[len(update.Message.Photo)-1]
	fileId := image.FileID

	file, err := bot.GetFile(fileId)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath)
	log.Printf("Image URL: %s", url)

	prompt := "Пожалуйста опишите изображение"
	if update.Message.Caption != "" {
		prompt = update.Message.Caption
	}

	messages := []gpt.Message{
		{Role: "user", Content: []gpt.Content{
			{Type: gpt.TypeText, Text: prompt},
			{Type: gpt.TypeImageUrl, ImageUrl: gpt.ImageUrl{Url: url}},
		}},
	}

	response := "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	responsePayload, err := gptClient.CallGPT(messages, gpt.ModelGPT4Vision, 0.8)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if len(responsePayload.Choices) > 0 {
		log.Print(responsePayload)
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	if chat.Settings.UseMarkdown {
		bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response)
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, response)
	}
}

func callCommand(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	command := update.Message.Command()

	if commands.CommandList[command] != nil {
		_command := commands.CommandList[command]
		if update.Message.From.ID == bot.AdminId || !_command.IsAdmin() {
			commands.CommandList[command].Execute(bot, update, gptClient, chat)
		}
	}
}

func callReply(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if chat.ChatID < 0 && update.Message.Voice == nil { // group chat
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
	responsePayload, err := gptClient.CallGPT(messages, chat.Settings.Model, chat.Settings.Temperature)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	// Add the assistant's response to the conversation history
	historyEntry.Response = storage.Message{Role: "assistant", Content: response}

	log.Printf("[%s] %s", "ChatGPT", response)
	if chat.Settings.UseMarkdown {
		bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response)
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, response)
	}

	// initial message was Voice
	if update.Message.Voice != nil {
		log.Print("Audio response")
		err = processVoice(bot, gptClient, chat.ChatID, response)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
	}

	if !util.IsIdInList(update.Message.From.ID, bot.Config.IgnoreReportIds) {
		bot.Log(fmt.Sprintf("[%s]\nMessage: %s\nResponse: %s", chat.Title, update.Message.Text, response))
	}
}

func processAudio(bot *telegram.Bot, gptClient *gpt.GPTClient, fileID string) (string, error) {
	// Download the voice message file
	file, err := bot.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %w", err)
	}

	// Download the audio file content
	audioURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath)
	audioContent, err := util.DownloadFile(audioURL)
	if err != nil {
		return "", fmt.Errorf("error downloading audio file: %w", err)
	}

	return gptClient.TranscribeAudio(audioContent)
}

func processVoice(bot *telegram.Bot, gptClient *gpt.GPTClient, chatID int64, inputText string) error {
	bytes, err := gptClient.GenerateVoice(inputText, gpt.VoiceModel, gpt.VoiceOnyx)
	if err != nil {
		return err
	}

	return bot.AudioUpload(chatID, bytes)
}
