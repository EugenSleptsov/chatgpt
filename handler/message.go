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

func (m *MessageHandler) Match(_ *telegram.UpdateContext) bool {
	return true // catch-all: handles any text message
}

func (m *MessageHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) error {
	if ctx.IsGroup {
		return m.handleGroup(ctx, chat)
	}
	return m.handlePrivate(ctx, chat)
}

// handlePrivate handles messages in 1:1 private chats — full GPT response every time.
func (m *MessageHandler) handlePrivate(ctx *telegram.UpdateContext, chat *storage.Chat) error {
	response, err := m.Deps.GPTService.ChatCompletion(chat, ctx.Text)
	m.Deps.Notifier.LogError(err)
	m.Deps.Notifier.Logf("[%s] %s", "ChatGPT", response)
	m.Deps.Bot.ReplyMarkdown(chat.ChatID, ctx.MessageID, response, chat.Settings.UseMarkdown)
	m.Deps.Notifier.ReportAdmin(ctx.SenderID, fmt.Sprintf("[%s | %s]\nMessage: %s\nResponse: %s", chat.Title, chat.ActiveSession().Model, ctx.Text, response))
	return nil
}

// handleGroup handles messages in group/supergroup chats.
// Every message is logged for context. Bot replies only when mentioned/replied-to,
// or when auto-reply decides to join in.
func (m *MessageHandler) handleGroup(ctx *telegram.UpdateContext, chat *storage.Chat) error {
	text := ctx.Text
	if text == "" {
		return nil
	}

	author := ctx.SenderName
	botUsername := m.Deps.Bot.GetUsername()
	botMentioned := strings.Contains(text, "@"+botUsername)
	botCalledByName := strings.Contains(strings.ToLower(text), "бот")
	isReplyToBot := ctx.Msg.ReplyToMessage != nil &&
		ctx.Msg.ReplyToMessage.From != nil &&
		ctx.Msg.ReplyToMessage.From.UserName == botUsername

	// Clean mention from text before storing
	cleanText := text
	if botMentioned {
		cleanText = strings.TrimSpace(strings.ReplaceAll(text, "@"+botUsername, ""))
	}

	// Always log message for context, regardless of who sent it
	m.Deps.GPTService.LogGroupMessage(chat, author, cleanText)

	// Edited messages: just update context, don't trigger responses.
	if ctx.IsEdited {
		m.Deps.Notifier.Logf("[Group] %s → отредактировано, обновляю контекст", author)
		return nil
	}

	// Bot explicitly addressed → respond
	if botMentioned || isReplyToBot || botCalledByName {
		m.Deps.Notifier.Logf("[Group] %s → бот упомянут, отвечаю", author)
		return m.replyToGroup(ctx, chat)
	}

	// Auto-reply: check only on authorized users' messages to avoid
	// burning GPT calls for every message in a busy group.
	if chat.Settings.GroupAutoReply && m.Deps.Auth.IsAuthorized(ctx.SenderID) {
		should, reason, err := m.Deps.GPTService.ShouldAutoReply(chat)
		m.Deps.Notifier.LogError(err)
		if should {
			m.Deps.Notifier.Logf("[Group] Авто-ответ: ДА (%s)", reason)
			return m.replyToGroup(ctx, chat)
		}
		m.Deps.Notifier.Logf("[Group] Авто-ответ: НЕТ (%s)", reason)
	}

	return nil
}

// replyToGroup calls GPT with full group history and sends the response.
func (m *MessageHandler) replyToGroup(ctx *telegram.UpdateContext, chat *storage.Chat) error {
	response, err := m.Deps.GPTService.ReplyFromGroupHistory(chat)
	m.Deps.Notifier.LogError(err)
	m.Deps.Notifier.Logf("[GroupGPT] %s", response)
	m.Deps.Bot.ReplyMarkdown(chat.ChatID, ctx.MessageID, response, chat.Settings.UseMarkdown)
	return nil
}
