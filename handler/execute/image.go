package execute

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
)

// ImageAnalysisExecutor handles IntentAnalyzeImage — sends an uploaded
// image to GPT Vision and returns the description.
type ImageAnalysisExecutor struct {
	Deps *commands.Deps
}

func (e *ImageAnalysisExecutor) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *handler.Request) []handler.Response {
	prompt := req.Text
	if prompt == "" {
		prompt = "Пожалуйста, опишите изображение"
	}

	response, err := e.Deps.GPTService.AnalyzeImage(req.ImageURL, prompt)
	if err != nil {
		e.Deps.Notifier.LogError(err)
		return []handler.Response{{Text: "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"}}
	}

	if ctx.IsGroup {
		e.Deps.GPTService.LogGroupMessage(chat, ctx.SenderName, fmt.Sprintf("[Фото] %s", prompt))
		e.Deps.GPTService.LogBotResponse(chat, response)
	}

	return []handler.Response{{Text: response, Markdown: chat.Settings.UseMarkdown}}
}
