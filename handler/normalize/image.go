package normalize

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/storage"
	"strings"
)

type ImageHandler struct {
	Deps *commands.Deps
}

func (i *ImageHandler) Match(ctx *telegram.UpdateContext) bool {
	return ctx.IsPhoto
}

// Handle resolves the image URL and extracts caption, producing a normalized Request.
func (i *ImageHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) *handler.Request {
	image := ctx.Msg.Photo[len(ctx.Msg.Photo)-1]
	file, err := i.Deps.Bot.GetFile(image.FileID)
	if err != nil {
		i.Deps.Notifier.LogError(err)
		return nil
	}

	imageURL := i.Deps.Bot.FileURL(file.FilePath)
	caption := ctx.Msg.Caption
	botAddressed := false

	if ctx.IsGroup {
		botUsername := i.Deps.Bot.GetUsername()
		botMentioned := strings.Contains(caption, "@"+botUsername)
		isReplyToBot := ctx.Msg.ReplyToMessage != nil &&
			ctx.Msg.ReplyToMessage.From != nil &&
			ctx.Msg.ReplyToMessage.From.UserName == botUsername
		botAddressed = botMentioned || isReplyToBot

		if botMentioned {
			caption = strings.TrimSpace(strings.ReplaceAll(caption, "@"+botUsername, ""))
		}

		i.Deps.GPTService.LogGroupPhoto(chat, ctx.SenderName, ctx.Msg.Caption)
		i.Deps.Notifier.Logf("[Group] %s → фото, botAddressed=%v", ctx.SenderName, botAddressed)
	}

	return &handler.Request{
		Text:          caption,
		ImageURL:      imageURL,
		BotAddressed:  botAddressed,
		OriginalMedia: handler.MediaImage,
	}
}
