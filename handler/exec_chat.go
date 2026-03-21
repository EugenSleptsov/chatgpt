package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
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

func (e *ChatExecutor) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Response {
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
	if req.OriginalMedia == MediaVoice && result.Audio == nil {
		audio, voiceErr := e.Deps.GPTService.GenerateVoice(result.Text)
		e.Deps.Notifier.LogError(voiceErr)
		if audio != nil {
			responses = append(responses, Response{Audio: audio})
		}
	}

	return responses
}

// chatResultToResponses maps a service.ChatResult to handler.Response slice.
func chatResultToResponses(r *service.ChatResult, markdown bool) []Response {
	var out []Response

	if r.Text != "" {
		out = append(out, Response{Text: r.Text, Markdown: markdown})
	}
	for _, img := range r.Images {
		out = append(out, Response{ImageURL: img.URL, Caption: img.Caption})
	}
	if r.Audio != nil {
		out = append(out, Response{Audio: r.Audio})
	}

	return out
}
