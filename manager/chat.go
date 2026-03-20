// Package manager handles Telegram-specific chat lifecycle:
// creating/finding chat objects in storage and writing chat message logs.
// It depends on Telegram types (Update) by design.
package manager

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type ChatManager interface {
	GetOrCreateChat(update telegram.Update) *storage.Chat
	LogMessage(update telegram.Update, chat *storage.Chat)
	Save()
	MarkDirty(chatID int64)
}
