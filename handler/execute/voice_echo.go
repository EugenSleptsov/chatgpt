package execute

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
)

// EchoTranscriptionExecutor handles IntentEchoTranscription — sends the
// voice transcription back to the user as a text message.
type EchoTranscriptionExecutor struct {
	Deps *commands.Deps
}

func (e *EchoTranscriptionExecutor) Execute(ctx *telegram.UpdateContext, _ *storage.Chat, req *handler.Request) []handler.Response {
	if req.IsForwarded {
		e.Deps.Notifier.Notify(fmt.Sprintf("[%s] Transcribe was done", ctx.ChatTitle()))
	}
	return []handler.Response{{Text: req.Text}}
}
