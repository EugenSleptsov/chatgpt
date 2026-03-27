package executor

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

// StickerExecutor logs stickers for group context. Nothing to process further.
type StickerExecutor struct {
	History  *service.HistoryService
	Notifier *service.Notifier
}

func (e *StickerExecutor) Match(ctx *pipeline.RequestContext) bool {
	return ctx.IsSticker
}

func (e *StickerExecutor) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if !ctx.IsGroup {
		return nil
	}

	e.History.LogGroupSticker(chat, ctx.SenderName, ctx.StickerEmoji)
	e.Notifier.Logf("[Group] %s → стикер %s, логирую", ctx.SenderName, ctx.StickerEmoji)
	return nil
}
