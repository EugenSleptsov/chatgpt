package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strings"
	"time"
)

type Worker struct {
	TelegramClient *telegram.Bot
	GptClient      *gpt.GPTClient
	StorageClient  storage.Storage
	LogClient      *log.Log
}

func NewWorker(telegramClient *telegram.Bot, gptClient *gpt.GPTClient, storageClient storage.Storage, logClient *log.Log) *Worker {
	return &Worker{
		TelegramClient: telegramClient,
		GptClient:      gptClient,
		StorageClient:  storageClient,
		LogClient:      logClient,
	}
}

func (w *Worker) Start(updateChan <-chan telegram.Update) {
	for update := range updateChan {
		w.ProcessUpdate(update)
		w.StorageClient.Save()
	}
}

func (w *Worker) LogMessage(update telegram.Update, chat *storage.Chat) {
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

	w.LogClient.LogToFile(fmt.Sprintf("log/%d.log", chat.ChatID), lines)
}

func (w *Worker) ProcessUpdate(update telegram.Update) {
	// Ignore any non-Message Updates
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	chat, ok := w.StorageClient.Get(chatID)
	if !ok {
		chat = createNewChat(update, w.TelegramClient)
		_ = w.StorageClient.Set(chatID, chat)
	}
	chat.Title = telegram.GetChatTitle(update)

	if !update.Message.IsCommand() {
		w.LogMessage(update, chat)
	}

	// If no authorized users are provided, make the bot public
	if !w.TelegramClient.IsAuthorizedUser(update.Message.From.ID) {
		if update.Message.Chat.Type == "private" {
			w.TelegramClient.Reply(chatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
			w.TelegramClient.Log(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, update.Message.Text))
		}
		return
	}

	if update.Message.Voice != nil {
		response, err := w.processAudio(update.Message.Voice.FileID)
		w.LogClient.LogError(err)
		w.TelegramClient.Reply(chatID, update.Message.MessageID, response)

		// check if message is forwarded, then we finish here
		if update.Message.ForwardFrom != nil {
			w.TelegramClient.Log(fmt.Sprintf("[%s] %s", telegram.GetChatTitle(update), "Transcribe was done"))
			return
		}
		update.Message.Text = response
	}

	if len(update.Message.Photo) > 0 {
		w.CallImageReply(update, chat)
		return
	}

	// Check for commands
	if update.Message.IsCommand() {
		w.CallCommand(update, chat)
	} else {
		w.CallReply(update, chat)
	}
}

func (w *Worker) CallImageReply(update telegram.Update, chat *storage.Chat) {
	image := update.Message.Photo[len(update.Message.Photo)-1]
	fileId := image.FileID

	file, err := w.TelegramClient.GetFile(fileId)
	w.LogClient.LogError(err)

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", w.TelegramClient.Token, file.FilePath)
	w.LogClient.LogSystemF("Image URL: %s", url)

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
	responsePayload, err := w.GptClient.CallGPT(messages, gpt.ModelGPT4Vision, 0.8)
	w.LogClient.LogError(err)

	if len(responsePayload.Choices) > 0 {
		w.LogClient.LogSystem(responsePayload)
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	w.TelegramClient.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
}

func createNewChat(update telegram.Update, bot *telegram.Bot) *storage.Chat {
	return &storage.Chat{
		ChatID: update.Message.Chat.ID,
		Settings: storage.ChatSettings{
			Temperature:     0.8,
			Model:           gpt.ModelGPT4OmniMini,
			MaxMessages:     bot.Config.MaxMessages,
			UseMarkdown:     true,
			SystemPrompt:    "You are a helpful ChatGPT bot based on OpenAI GPT Language model. You are a helpful assistant that always tries to help and answer with relevant information as possible.",
			SummarizePrompt: bot.Config.SummarizePrompt,
			Token:           bot.Config.GPTToken,
		},
		History:          make([]*storage.ConversationEntry, 0),
		ImageGenNextTime: time.Now(),
		Title:            telegram.GetChatTitle(update),
	}
}

func (w *Worker) CallCommand(update telegram.Update, chat *storage.Chat) {
	command := update.Message.Command()

	if cmd, exists := commands.CommandList[command]; exists {
		if update.Message.From.ID == w.TelegramClient.AdminId || !cmd.IsAdmin() {
			cmd.Execute(w.TelegramClient, update, w.GptClient, chat)
		}
	}
}

func (w *Worker) CallReply(update telegram.Update, chat *storage.Chat) {
	if chat.ChatID < 0 && update.Message.Voice == nil { // group chat
		isReplyToBot := update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == w.TelegramClient.Username
		if !strings.Contains(update.Message.Text, "@"+w.TelegramClient.Username) && !isReplyToBot {
			return
		}

		if strings.Contains(update.Message.Text, "@"+w.TelegramClient.Username) {
			update.Message.Text = strings.Replace(update.Message.Text, "@"+w.TelegramClient.Username, "", -1)
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
	responsePayload, err := w.GptClient.CallGPT(messages, chat.Settings.Model, chat.Settings.Temperature)
	w.LogClient.LogError(err)

	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	// Add the assistant's response to the conversation history
	historyEntry.Response = storage.Message{Role: "assistant", Content: response}
	w.LogClient.LogSystemF("[%s] %s", "ChatGPT", response)
	w.TelegramClient.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)

	// initial message was Voice
	if update.Message.Voice != nil {
		w.LogClient.LogSystem("Audio response")

		bytes, err := w.GptClient.GenerateVoice(response, gpt.VoiceModel, gpt.VoiceOnyx)
		w.LogClient.LogError(err)
		err = w.TelegramClient.AudioUpload(chat.ChatID, bytes)
		w.LogClient.LogError(err)
	}

	w.TelegramClient.ReportAdmin(update.Message.From.ID, fmt.Sprintf("[%s | %s]\nMessage: %s\nResponse: %s", chat.Title, chat.Settings.Model, update.Message.Text, response))
}

func (w *Worker) processAudio(fileID string) (string, error) {
	// Download the voice message file
	file, err := w.TelegramClient.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %w", err)
	}

	// Download the audio file content
	audioURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", w.TelegramClient.Token, file.FilePath)
	audioContent, err := util.DownloadFile(audioURL)
	if err != nil {
		return "", fmt.Errorf("error downloading audio file: %w", err)
	}

	return w.GptClient.TranscribeAudio(audioContent)
}
