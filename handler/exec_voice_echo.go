package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"fmt"
)

// EchoTranscriptionExecutor handles IntentEchoTranscription — sends the
// voice transcription back to the user as a text message.
type EchoTranscriptionExecutor struct {
	Deps *commands.Deps
}

func (e *EchoTranscriptionExecutor) Execute(ctx *telegram.UpdateContext, _ *storage.Chat, req *Request) []Response {
	if req.IsForwarded {
		e.Deps.Notifier.Notify(fmt.Sprintf("[%s] Transcribe was done", ctx.ChatTitle()))
	}
	return []Response{{Text: req.Text}}
}
