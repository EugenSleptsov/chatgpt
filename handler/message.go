package handler

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type MessageHandler struct {
	Deps *commands.Deps
}

func (m *MessageHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	// Filter group chats: only respond when explicitly mentioned or replied to
	if chat.ChatID < 0 && update.Message.Voice == nil {
		isReplyToBot := update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == m.Deps.Bot.Username
		if !strings.Contains(update.Message.Text, "@"+m.Deps.Bot.Username) && !isReplyToBot {
			return nil
		}
		update.Message.Text = strings.ReplaceAll(update.Message.Text, "@"+m.Deps.Bot.Username, "")
	}

	// Business logic — single service call
	response := m.Deps.ChatService.ChatCompletion(chat, update.Message.Text)

	// Presentation
	m.Deps.Log.Logf("[%s] %s", "ChatGPT", response)
	m.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)

	// Voice reply if the original message was voice
	if update.Message.Voice != nil {
		m.Deps.Log.Log("Audio response")
		bytes, err := m.Deps.GptClient.GenerateVoice(response, gpt.VoiceModel, gpt.VoiceOnyx)
		m.Deps.ErrorLog.LogError(err)
		err = m.Deps.Bot.AudioUpload(chat.ChatID, bytes)
		m.Deps.ErrorLog.LogError(err)
	}

	m.Deps.Bot.ReportAdmin(update.Message.From.ID, fmt.Sprintf("[%s | %s]\nMessage: %s\nResponse: %s", chat.Title, chat.Settings.Model, update.Message.Text, response))
	return nil
}
