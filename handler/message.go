package handler

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type MessageHandler struct {
	TelegramClient *telegram.Bot
	GptClient      gpt.Client
	LogClient      log.Log
	ErrorLogClient log.ErrorLog
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
	m.ErrorLogClient.LogError(err)

	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	// Add the assistant's response to the conversation history
	historyEntry.Response = storage.Message{Role: "assistant", Content: response}
	m.LogClient.Logf("[%s] %s", "ChatGPT", response)
	m.TelegramClient.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)

	// initial message was Voice
	if update.Message.Voice != nil {
		m.LogClient.Log("Audio response")

		bytes, err := m.GptClient.GenerateVoice(response, gpt.VoiceModel, gpt.VoiceOnyx)
		m.ErrorLogClient.LogError(err)
		err = m.TelegramClient.AudioUpload(chat.ChatID, bytes)
		m.ErrorLogClient.LogError(err)
	}

	m.TelegramClient.ReportAdmin(update.Message.From.ID, fmt.Sprintf("[%s | %s]\nMessage: %s\nResponse: %s", chat.Title, chat.Settings.Model, update.Message.Text, response))
	return nil
}
