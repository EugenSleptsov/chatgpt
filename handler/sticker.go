package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
)

type StickerHandler struct {
	Deps *commands.Deps
}

func (s *StickerHandler) Match(ctx *telegram.UpdateContext) bool {
	return ctx.IsSticker
}

// Handle logs stickers as text placeholders in group chats for conversation context.
// In private chats, stickers are silently ignored.
func (s *StickerHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) error {
	if !ctx.IsGroup {
		return nil
	}

	emoji := ""
	if ctx.Msg.Sticker != nil {
		emoji = ctx.Msg.Sticker.Emoji
	}
	s.Deps.GPTService.LogGroupSticker(chat, ctx.SenderName, emoji)
	s.Deps.Notifier.Logf("[Group] %s → стикер %s, логирую", ctx.SenderName, emoji)
	return nil
}
