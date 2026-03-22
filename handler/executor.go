package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

// IntentExecutor handles a single intent type and produces responses.
type IntentExecutor interface {
	Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Response
}
