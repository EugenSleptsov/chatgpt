package normalize

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/storage"
	"strings"
)

type MessageHandler struct {
	Deps *commands.Deps
}

func (m *MessageHandler) Match(_ *telegram.UpdateContext) bool {
	return true // catch-all
}

// Handle normalizes a text message into a Request.
// In groups: logs for context, detects mentions, cleans text.
// Returns nil for edited messages (context-only) and empty text.
func (m *MessageHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) *handler.Request {
	text := ctx.Text
	if text == "" {
		return nil
	}

	if !ctx.IsGroup {
		return &handler.Request{Text: text}
	}

	// --- Group normalization ---
	botUsername := m.Deps.Bot.GetUsername()
	botMentioned := strings.Contains(text, "@"+botUsername)
	botCalledByName := strings.Contains(strings.ToLower(text), "бот")
	isReplyToBot := ctx.Msg.ReplyToMessage != nil &&
		ctx.Msg.ReplyToMessage.From != nil &&
		ctx.Msg.ReplyToMessage.From.UserName == botUsername

	cleanText := text
	if botMentioned {
		cleanText = strings.TrimSpace(strings.ReplaceAll(text, "@"+botUsername, ""))
	}

	// Always log message for group context
	m.Deps.GPTService.LogGroupMessage(chat, ctx.SenderName, cleanText)

	// Edited messages: just updated context, nothing to process
	if ctx.IsEdited {
		m.Deps.Notifier.Logf("[Group] %s → отредактировано, обновляю контекст", ctx.SenderName)
		return nil
	}

	return &handler.Request{
		Text:         cleanText,
		BotAddressed: botMentioned || isReplyToBot || botCalledByName,
	}
}
