package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"fmt"
)

// ImageAnalysisExecutor handles IntentAnalyzeImage — sends an image to
// GPT Vision and returns the description.
type ImageAnalysisExecutor struct {
	Deps *commands.Deps
}

func (e *ImageAnalysisExecutor) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Response {
	prompt := req.Text
	if prompt == "" {
		prompt = "Пожалуйста, опишите изображение"
	}

	response, err := e.Deps.GPTService.AnalyzeImage(req.ImageURL, prompt)
	if err != nil {
		e.Deps.Notifier.LogError(err)
		return []Response{{Text: "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"}}
	}

	if ctx.IsGroup {
		e.Deps.GPTService.LogGroupMessage(chat, ctx.SenderName, fmt.Sprintf("[Фото] %s", prompt))
		e.Deps.GPTService.LogBotResponse(chat, response)
	}

	return []Response{{Text: response, Markdown: chat.Settings.UseMarkdown}}
}
