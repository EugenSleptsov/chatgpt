package handler

import (
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
		isReplyToBot := update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == m.Deps.Bot.GetUsername()
		if !strings.Contains(update.Message.Text, "@"+m.Deps.Bot.GetUsername()) && !isReplyToBot {
			return nil
		}
		update.Message.Text = strings.ReplaceAll(update.Message.Text, "@"+m.Deps.Bot.GetUsername(), "")
	}

	// Business logic — single service call
	response, err := m.Deps.GPTService.ChatCompletion(chat, update.Message.Text)
	m.Deps.Notifier.LogError(err)

	// Presentation
	m.Deps.Notifier.Logf("[%s] %s", "ChatGPT", response)
	m.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)

	// Voice reply if the original message was voice
	if update.Message.Voice != nil {
		m.Deps.Notifier.Logf("Audio response")
		bytes, err := m.Deps.GPTService.GenerateVoice(response)
		m.Deps.Notifier.LogError(err)
		err = m.Deps.Bot.AudioUpload(chat.ChatID, bytes)
		m.Deps.Notifier.LogError(err)
	}

	m.Deps.Notifier.ReportAdmin(update.Message.From.ID, fmt.Sprintf("[%s | %s]\nMessage: %s\nResponse: %s", chat.Title, chat.ActiveSession().Model, update.Message.Text, response))
	return nil
}
