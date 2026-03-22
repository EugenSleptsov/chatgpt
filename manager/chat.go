// Package manager handles Telegram-specific chat lifecycle:
// creating/finding chat objects in storage and writing chat message logs.
// It depends on Telegram types (UpdateContext) by design.
package manager

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type ChatManager interface {
	GetOrCreateChat(ctx *telegram.UpdateContext) *storage.Chat
	LogMessage(ctx *telegram.UpdateContext, chat *storage.Chat)
	Save()
	MarkDirty(chatID int64)
}
