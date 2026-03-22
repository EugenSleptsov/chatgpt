package execute

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/service"
	"GPTBot/storage"
	"fmt"
)

// ChatExecutor handles IntentChat — private chat completion via GPT.
// GPT may call function tools (generate_image, generate_voice); the
// executor collects all results into the response list.
type ChatExecutor struct {
	Deps *commands.Deps
}

func (e *ChatExecutor) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *handler.Request) []handler.Response {
	result, err := e.Deps.GPTService.ChatCompletion(chat, req.Text)
	e.Deps.Notifier.LogError(err)

	if result.Text != "" {
		e.Deps.Notifier.Logf("[ChatGPT] %s", result.Text)
	}
	e.Deps.Notifier.ReportAdmin(ctx.SenderID, fmt.Sprintf(
		"[%s | %s]\nMessage: %s\nResponse: %s\nImages: %d, Audio: %v",
		chat.Title, chat.ActiveSession().Model, req.Text, result.Text,
		len(result.Images), result.Audio != nil,
	))

	responses := chatResultToResponses(result, chat.Settings.UseMarkdown)

	// Voice-input guarantee: if the user sent voice and GPT didn't call
	// generate_voice, we synthesize audio from the text response.
	if req.OriginalMedia == handler.MediaVoice && result.Audio == nil {
		audio, voiceErr := e.Deps.GPTService.GenerateVoice(result.Text)
		e.Deps.Notifier.LogError(voiceErr)
		if audio != nil {
			responses = append(responses, handler.Response{Audio: audio})
		}
	}

	return responses
}

// chatResultToResponses maps a service.ChatResult to []handler.Response.
// Used by all group/private executors.
//
// When images are present the model's text reply becomes the caption of the
// first image instead of being sent as a separate message — this avoids the
// user seeing the raw DALL-E prompt as a caption.
func chatResultToResponses(r *service.ChatResult, markdown bool) []handler.Response {
	var out []handler.Response

	textUsed := false
	for i, img := range r.Images {
		caption := ""
		if i == 0 && r.Text != "" {
			caption = r.Text
			textUsed = true
		}
		out = append(out, handler.Response{ImageData: img.Data, Caption: caption})
	}

	if r.Text != "" && !textUsed {
		out = append(out, handler.Response{Text: r.Text, Markdown: markdown})
	}
	if r.Audio != nil {
		out = append(out, handler.Response{Audio: r.Audio})
	}

	return out
}
