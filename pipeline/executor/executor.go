package executor

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

// Executor handles a specific type of Telegram update end-to-end:
// from raw update context to final responses ready for delivery.
// The Decoder picks the first executor whose Match returns true,
// then calls Execute to produce the response list.
type Executor interface {
	Match(ctx *pipeline.RequestContext) bool
	Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response
}
