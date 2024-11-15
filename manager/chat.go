package manager

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type ChatManager interface {
	GetStorageClient() storage.Storage
	GetOrCreateChat(update telegram.Update) *storage.Chat
	LogMessage(update telegram.Update, chat *storage.Chat)
}
