package handler

import "GPTBot/commands"

// Handlers returns all available handler constructors.
// To add a new handler, append it here — no changes in main.go needed.
func Handlers() []func(d *commands.Deps) UpdateHandler {
	return []func(d *commands.Deps) UpdateHandler{
		func(d *commands.Deps) UpdateHandler { return &CommandHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &VoiceHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &ImageHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &StickerHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &MessageHandler{Deps: d} }, // catch-all, must be last by priority
	}
}

// NewRouter creates a Router with all handlers registered.
func NewRouter(deps *commands.Deps) *Router {
	r := &Router{}
	for _, ctor := range Handlers() {
		r.Register(ctor(deps))
	}
	return r
}
