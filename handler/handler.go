package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type UpdateHandler interface {
	Handle(update telegram.Update, chat *storage.Chat) error
}
