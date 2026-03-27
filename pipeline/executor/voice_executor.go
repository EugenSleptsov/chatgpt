package executor

import (
	"GPTBot/application/service"
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/infrastructure/util"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

// VoiceExecutor transcribes voice messages and decides what to do next:
//   - forwarded → return transcription only
//   - not forwarded → echo transcription + process as text
type VoiceExecutor struct {
	Files        pipeline.FileResolver
	AIClient     ai.Client
	Notifier     *service.Notifier
	TextExecutor *TextExecutor
}

func (e *VoiceExecutor) Match(ctx *pipeline.RequestContext) bool {
	return ctx.IsVoice && !ctx.IsEdited
}

func (e *VoiceExecutor) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	transcription, err := e.transcribe(ctx.VoiceFileID)
	if err != nil {
		e.Notifier.LogError(err)
		return []sender.Response{{Text: "Не удалось обработать голосовое сообщение."}}
	}

	// Always echo transcription
	responses := []sender.Response{{Text: transcription}}

	if ctx.IsForwarded {
		e.Notifier.Notify(fmt.Sprintf("[%s] Transcribe was done", ctx.ChatTitle))
		return responses // forwarded → transcription only
	}

	// Not forwarded → also process as text (with voice flag for audio generation)
	responses = append(responses, e.TextExecutor.ProcessText(ctx, chat, transcription, true)...)
	return responses
}

func (e *VoiceExecutor) transcribe(fileID string) (string, error) {
	file, err := e.Files.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %w", err)
	}

	audioContent, err := util.DownloadFile(e.Files.FileURL(file.FilePath))
	if err != nil {
		return "", fmt.Errorf("error downloading audio file: %w", err)
	}

	return e.AIClient.TranscribeAudio(audioContent)
}
