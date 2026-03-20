package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
)

type StickerHandler struct {
	Deps *commands.Deps
}

// Handle logs stickers as text placeholders in group chats for conversation context.
// In private chats, stickers are silently ignored.
func (s *StickerHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	if chat.ChatID >= 0 {
		return nil
	}

	author := authorName(update)
	emoji := ""
	if update.Message.Sticker != nil {
		emoji = update.Message.Sticker.Emoji
	}
	s.Deps.GPTService.LogGroupSticker(chat, author, emoji)
	s.Deps.Notifier.Logf("[Group] %s → стикер %s, логирую", author, emoji)
	return nil
}
