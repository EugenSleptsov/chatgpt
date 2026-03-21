package telegram

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UpdateContext is a pre-parsed, handler-friendly view of a Telegram update.
// It extracts the effective message and pre-computes commonly used fields,
// eliminating the need to call update.Msg() and nil-check everywhere.
type UpdateContext struct {
	Update Update            // original update for edge-case access
	Msg    *tgbotapi.Message // effective message — never nil after successful construction

	ChatID     int64
	MessageID  int
	SenderID   int64
	SenderName string
	Text       string // Msg.Text or Msg.Caption, whichever is non-empty

	IsEdited  bool
	IsGroup   bool
	IsCommand bool
	IsPhoto   bool
	IsVoice   bool
	IsSticker bool
}

// NewUpdateContext builds an UpdateContext from a raw Update.
// Returns nil if the update contains no effective message
// (e.g. callback_query, inline_query, poll — not yet supported).
func NewUpdateContext(update Update) *UpdateContext {
	msg := update.Msg()
	if msg == nil {
		return nil
	}

	ctx := &UpdateContext{
		Update:    update,
		Msg:       msg,
		ChatID:    msg.Chat.ID,
		MessageID: msg.MessageID,
		IsEdited:  update.IsEdited(),
		IsGroup:   msg.Chat.ID < 0,
	}

	if msg.From != nil {
		ctx.SenderID = msg.From.ID
		ctx.SenderName = strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName)
		if ctx.SenderName == "" {
			ctx.SenderName = msg.From.UserName
		}
	}

	ctx.Text = msg.Text
	if ctx.Text == "" {
		ctx.Text = msg.Caption
	}

	ctx.IsCommand = msg.IsCommand()
	ctx.IsPhoto = len(msg.Photo) > 0
	ctx.IsVoice = msg.Voice != nil
	ctx.IsSticker = msg.Sticker != nil

	return ctx
}

// ChatTitle returns a human-readable title for the chat.
func (ctx *UpdateContext) ChatTitle() string {
	if ctx.ChatID > 0 {
		return fmt.Sprintf("%s %s [@%s / %d]",
			ctx.Msg.Chat.FirstName, ctx.Msg.Chat.LastName,
			ctx.Msg.Chat.UserName, ctx.ChatID)
	}
	return fmt.Sprintf("Chat %d [%s]", ctx.ChatID, ctx.Msg.Chat.Title)
}
