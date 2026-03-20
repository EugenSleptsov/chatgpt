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
	if chat.ChatID < 0 {
		return m.handleGroup(update, chat)
	}
	return m.handlePrivate(update, chat)
}

// handlePrivate handles messages in 1:1 private chats — full GPT response every time.
func (m *MessageHandler) handlePrivate(update telegram.Update, chat *storage.Chat) error {
	response, err := m.Deps.GPTService.ChatCompletion(chat, update.Message.Text)
	m.Deps.Notifier.LogError(err)
	m.Deps.Notifier.Logf("[%s] %s", "ChatGPT", response)
	m.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
	m.Deps.Notifier.ReportAdmin(update.Message.From.ID, fmt.Sprintf("[%s | %s]\nMessage: %s\nResponse: %s", chat.Title, chat.ActiveSession().Model, update.Message.Text, response))
	return nil
}

// handleGroup handles messages in group/supergroup chats.
// Every message is logged for context. Bot replies only when mentioned/replied-to,
// or when auto-reply decides to join in.
func (m *MessageHandler) handleGroup(update telegram.Update, chat *storage.Chat) error {
	text := update.Message.Text
	if text == "" {
		return nil
	}

	author := authorName(update)
	botUsername := m.Deps.Bot.GetUsername()
	botMentioned := strings.Contains(text, "@"+botUsername)
	botCalledByName := strings.Contains(strings.ToLower(text), "бот")
	isReplyToBot := update.Message.ReplyToMessage != nil &&
		update.Message.ReplyToMessage.From != nil &&
		update.Message.ReplyToMessage.From.UserName == botUsername

	// Clean mention from text before storing
	cleanText := text
	if botMentioned {
		cleanText = strings.TrimSpace(strings.ReplaceAll(text, "@"+botUsername, ""))
	}

	// Always log message for context, regardless of who sent it
	m.Deps.GPTService.LogGroupMessage(chat, author, cleanText)

	// Bot explicitly addressed → respond
	if botMentioned || isReplyToBot || botCalledByName {
		m.Deps.Notifier.Logf("[Group] %s → бот упомянут, отвечаю", author)
		return m.replyToGroup(update, chat)
	}

	// Auto-reply: check only on authorized users' messages to avoid
	// burning GPT calls for every message in a busy group.
	if chat.Settings.GroupAutoReply && m.Deps.Auth.IsAuthorized(update.Message.From.ID) {
		should, reason, err := m.Deps.GPTService.ShouldAutoReply(chat)
		m.Deps.Notifier.LogError(err)
		if should {
			m.Deps.Notifier.Logf("[Group] Авто-ответ: ДА (%s)", reason)
			return m.replyToGroup(update, chat)
		}
		m.Deps.Notifier.Logf("[Group] Авто-ответ: НЕТ (%s)", reason)
	}

	return nil
}

// replyToGroup calls GPT with full group history and sends the response.
func (m *MessageHandler) replyToGroup(update telegram.Update, chat *storage.Chat) error {
	response, err := m.Deps.GPTService.ReplyFromGroupHistory(chat)
	m.Deps.Notifier.LogError(err)
	m.Deps.Notifier.Logf("[GroupGPT] %s", response)
	m.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
	return nil
}
