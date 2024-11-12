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
)

type UpdateHandler interface {
	Handle(update telegram.Update, chat *storage.Chat) error
}

type UpdateHandlerFactory interface {
	GetHandler(update telegram.Update) UpdateHandler
}

type ConcreteUpdateHandlerFactory struct {
	TelegramBot    *telegram.Bot
	CommandFactory commands.CommandFactory
	GptClient      *gpt.GPTClient
	LogClient      *log.Log
}

func (c *ConcreteUpdateHandlerFactory) GetHandler(update telegram.Update) UpdateHandler {
	if update.Message.IsCommand() {
		return &CommandHandler{
			TelegramClient: c.TelegramBot,
			CommandFactory: c.CommandFactory,
		}
	}

	if update.Message.Voice != nil {
		return &VoiceHandler{
			TelegramClient: c.TelegramBot,
			GptClient:      c.GptClient,
			LogClient:      c.LogClient,
		}
	}

	if len(update.Message.Photo) > 0 {
		return &ImageHandler{
			TelegramClient: c.TelegramBot,
			GptClient:      c.GptClient,
			LogClient:      c.LogClient,
		}
	}

	return &MessageHandler{
		TelegramClient: c.TelegramBot,
		GptClient:      c.GptClient,
	}
}

type CommandHandler struct {
	TelegramClient *telegram.Bot
	CommandFactory commands.CommandFactory
}

func (c *CommandHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	command := update.Message.Command()

	cmd, err := c.CommandFactory.GetCommand(command)
	if err != nil {
		return err
	}

	if !cmd.IsAdmin() || update.Message.From.ID == c.TelegramClient.AdminId {
		cmd.Execute(update, chat)
	}

	return nil
}

type MessageHandler struct {
	TelegramClient *telegram.Bot
	GptClient      *gpt.GPTClient
	LogClient      *log.Log
}

func (m *MessageHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	if chat.ChatID < 0 && update.Message.Voice == nil { // group chat
		isReplyToBot := update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == m.TelegramClient.Username
		if !strings.Contains(update.Message.Text, "@"+m.TelegramClient.Username) && !isReplyToBot {
			return nil
		}

		if strings.Contains(update.Message.Text, "@"+m.TelegramClient.Username) {
			update.Message.Text = strings.Replace(update.Message.Text, "@"+m.TelegramClient.Username, "", -1)
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
	responsePayload, err := m.GptClient.CallGPT(messages, chat.Settings.Model, chat.Settings.Temperature)
	m.LogClient.LogError(err)

	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	// Add the assistant's response to the conversation history
	historyEntry.Response = storage.Message{Role: "assistant", Content: response}
	m.LogClient.LogSystemF("[%s] %s", "ChatGPT", response)
	m.TelegramClient.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)

	// initial message was Voice
	if update.Message.Voice != nil {
		m.LogClient.LogSystem("Audio response")

		bytes, err := m.GptClient.GenerateVoice(response, gpt.VoiceModel, gpt.VoiceOnyx)
		m.LogClient.LogError(err)
		err = m.TelegramClient.AudioUpload(chat.ChatID, bytes)
		m.LogClient.LogError(err)
	}

	m.TelegramClient.ReportAdmin(update.Message.From.ID, fmt.Sprintf("[%s | %s]\nMessage: %s\nResponse: %s", chat.Title, chat.Settings.Model, update.Message.Text, response))
	return nil
}

type VoiceHandler struct {
	TelegramClient *telegram.Bot
	GptClient      *gpt.GPTClient
	LogClient      *log.Log
}

func (v *VoiceHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	response, err := v.processAudio(update.Message.Voice.FileID)
	v.LogClient.LogError(err)
	v.TelegramClient.Reply(chat.ChatID, update.Message.MessageID, response)

	// check if message is forwarded, then we finish here
	if update.Message.ForwardFrom != nil {
		v.TelegramClient.Log(fmt.Sprintf("[%s] %s", telegram.GetChatTitle(update), "Transcribe was done"))
		return nil
	}
	update.Message.Text = response

	return nil
}

func (v *VoiceHandler) processAudio(fileID string) (string, error) {
	// Download the voice message file
	file, err := v.TelegramClient.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %w", err)
	}

	// Download the audio file content
	audioURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", v.TelegramClient.Token, file.FilePath)
	audioContent, err := util.DownloadFile(audioURL)
	if err != nil {
		return "", fmt.Errorf("error downloading audio file: %w", err)
	}

	return v.GptClient.TranscribeAudio(audioContent)
}

type ImageHandler struct {
	TelegramClient *telegram.Bot
	GptClient      *gpt.GPTClient
	LogClient      *log.Log
}

func (i *ImageHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	image := update.Message.Photo[len(update.Message.Photo)-1]
	fileId := image.FileID

	file, err := i.TelegramClient.GetFile(fileId)
	i.LogClient.LogError(err)

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", i.TelegramClient.Token, file.FilePath)
	i.LogClient.LogSystemF("Image URL: %s", url)

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
	responsePayload, err := i.GptClient.CallGPT(messages, gpt.ModelGPT4Vision, 0.8)
	i.LogClient.LogError(err)

	if len(responsePayload.Choices) > 0 {
		i.LogClient.LogSystem(responsePayload)
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	i.TelegramClient.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
	return nil
}
