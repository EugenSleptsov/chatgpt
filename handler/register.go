package handler

import "GPTBot/commands"

// Handlers returns all available handler constructors (normalization layer).
// Order matters: first match wins.
func Handlers() []func(d *commands.Deps) UpdateHandler {
	return []func(d *commands.Deps) UpdateHandler{
		func(d *commands.Deps) UpdateHandler { return &CommandHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &VoiceHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &ImageHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &StickerHandler{Deps: d} },
		func(d *commands.Deps) UpdateHandler { return &MessageHandler{Deps: d} }, // catch-all
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

// NewPipeline creates a Pipeline with resolver and all intent executors.
func NewPipeline(deps *commands.Deps) *Pipeline {
	p := &Pipeline{
		resolver:  &IntentResolver{},
		executors: make(map[IntentType]IntentExecutor),
	}

	p.RegisterExecutor(IntentChat, &ChatExecutor{Deps: deps})
	p.RegisterExecutor(IntentGroupReply, &GroupReplyExecutor{Deps: deps})
	p.RegisterExecutor(IntentGroupAutoReply, &GroupAutoReplyExecutor{Deps: deps})
	p.RegisterExecutor(IntentAnalyzeImage, &ImageAnalysisExecutor{Deps: deps})
	p.RegisterExecutor(IntentEchoTranscription, &EchoTranscriptionExecutor{Deps: deps})

	return p
}
