package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

// UpdateHandler handles a specific type of Telegram update.
type UpdateHandler interface {
	// Match reports whether this handler can process the given update.
	Match(ctx *telegram.UpdateContext) bool

	// Handle processes the update. Called only when Match returned true.
	Handle(ctx *telegram.UpdateContext, chat *storage.Chat) error
}
