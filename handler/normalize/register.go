package normalize

import (
	"GPTBot/commands"
	"GPTBot/handler"
)

// AllHandlers returns every normalize handler in priority order.
// The last entry (MessageHandler) is a catch-all and must stay last.
// To add a new handler, append it here — no other changes needed.
func AllHandlers(deps *commands.Deps) []handler.UpdateHandler {
	return []handler.UpdateHandler{
		&CommandHandler{},
		&VoiceHandler{Deps: deps},
		&ImageHandler{Deps: deps},
		&StickerHandler{Deps: deps},
		&MessageHandler{Deps: deps}, // catch-all — must be last
	}
}
