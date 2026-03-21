package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

// UpdateHandler normalizes a Telegram update into a Request.
// Returning nil means the handler dealt with it internally (commands, stickers)
// or there is nothing to process (edited messages, errors).
type UpdateHandler interface {
	Match(ctx *telegram.UpdateContext) bool
	Handle(ctx *telegram.UpdateContext, chat *storage.Chat) *Request
}
