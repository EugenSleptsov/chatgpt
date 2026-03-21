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

// Handle logs stickers for group context. Nothing to process further.
func (s *StickerHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) *Request {
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
