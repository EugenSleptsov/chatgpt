package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
)

type UpdateHandlerFactory interface {
	GetHandler(update telegram.Update) UpdateHandler
}

type ConcreteUpdateHandlerFactory struct {
	Deps *commands.Deps
}

func NewUpdateHandlerFactory(deps *commands.Deps) *ConcreteUpdateHandlerFactory {
	return &ConcreteUpdateHandlerFactory{Deps: deps}
}

func (c *ConcreteUpdateHandlerFactory) GetHandler(update telegram.Update) UpdateHandler {
	if update.Message.IsCommand() {
		return &CommandHandler{Deps: c.Deps}
	}

	if update.Message.Voice != nil {
		return &VoiceHandler{Deps: c.Deps}
	}

	if len(update.Message.Photo) > 0 {
		return &ImageHandler{Deps: c.Deps}
	}

	if update.Message.Sticker != nil {
		return &StickerHandler{Deps: c.Deps}
	}

	return &MessageHandler{Deps: c.Deps}
}
