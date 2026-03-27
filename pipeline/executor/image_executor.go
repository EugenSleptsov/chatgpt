package executor

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

// ImageExecutor resolves image URLs, detects group mentions,
// and runs GPT Vision analysis.
type ImageExecutor struct {
	Files       pipeline.FileResolver
	BotUsername string
	Commands    *service.GPTCommandService
	History     *service.HistoryService
	Notifier    *service.Notifier
}

func (e *ImageExecutor) Match(ctx *pipeline.RequestContext) bool {
	return ctx.IsPhoto
}

func (e *ImageExecutor) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	file, err := e.Files.GetFile(ctx.PhotoFileID)
	if err != nil {
		e.Notifier.LogError(err)
		return nil
	}

	imageURL := e.Files.FileURL(file.FilePath)
	caption := ctx.Caption

	if ctx.IsGroup {
		botMentioned := strings.Contains(caption, "@"+e.BotUsername)
		isReplyToBot := ctx.ReplyToUsername == e.BotUsername
		botAddressed := botMentioned || isReplyToBot

		if botMentioned {
			caption = strings.TrimSpace(strings.ReplaceAll(caption, "@"+e.BotUsername, ""))
		}

		e.History.LogGroupPhoto(chat, ctx.SenderName, ctx.Caption)
		e.Notifier.Logf("[Group] %s → фото, botAddressed=%v", ctx.SenderName, botAddressed)

		if !botAddressed {
			return nil
		}
	}

	prompt := caption
	if prompt == "" {
		prompt = "Пожалуйста, опишите изображение"
	}

	response, err := e.Commands.AnalyzeImage(imageURL, prompt)
	if err != nil {
		e.Notifier.LogError(err)
		return []sender.Response{{Text: "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"}}
	}

	if ctx.IsGroup {
		e.History.LogGroupMessage(chat, ctx.SenderName, fmt.Sprintf("[Фото] %s", caption))
		e.History.LogBotResponse(chat, response)
	}

	return []sender.Response{{Text: response, Markdown: chat.Settings.UseMarkdown}}
}
