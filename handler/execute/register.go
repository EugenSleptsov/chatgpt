package execute

import (
	"GPTBot/commands"
	"GPTBot/handler"
)

// AllExecutors returns every intent executor keyed by its IntentType.
// To add a new executor, append it here — no other changes needed.
func AllExecutors(deps *commands.Deps) map[handler.IntentType]handler.IntentExecutor {
	return map[handler.IntentType]handler.IntentExecutor{
		handler.IntentChat:              &ChatExecutor{Deps: deps},
		handler.IntentGroupReply:        &GroupReplyExecutor{Deps: deps},
		handler.IntentGroupAutoReply:    &GroupAutoReplyExecutor{Deps: deps},
		handler.IntentAnalyzeImage:      &ImageAnalysisExecutor{Deps: deps},
		handler.IntentEchoTranscription: &EchoTranscriptionExecutor{Deps: deps},
	}
}
