package handler

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/commands"
)

type UpdateHandlerFactory interface {
	GetHandler(update telegram.Update) UpdateHandler
}

type ConcreteUpdateHandlerFactory struct {
	TelegramBot    *telegram.Bot
	CommandFactory commands.CommandFactory
	GptClient      *gpt.GPTClient
	LogClient      *log.Log
}

func NewUpdateHandlerFactory(telegramBot *telegram.Bot, commandFactory commands.CommandFactory, gptClient *gpt.GPTClient, logClient *log.Log) *ConcreteUpdateHandlerFactory {
	return &ConcreteUpdateHandlerFactory{
		TelegramBot:    telegramBot,
		CommandFactory: commandFactory,
		GptClient:      gptClient,
		LogClient:      logClient,
	}
}

func (c *ConcreteUpdateHandlerFactory) GetHandler(update telegram.Update) UpdateHandler {
	if update.Message.IsCommand() {
		return &CommandHandler{
			TelegramClient: c.TelegramBot,
			CommandFactory: c.CommandFactory,
		}
	}

	if update.Message.Voice != nil {
		return &VoiceHandler{
			TelegramClient: c.TelegramBot,
			GptClient:      c.GptClient,
			LogClient:      c.LogClient,
		}
	}

	if len(update.Message.Photo) > 0 {
		return &ImageHandler{
			TelegramClient: c.TelegramBot,
			GptClient:      c.GptClient,
			LogClient:      c.LogClient,
		}
	}

	return &MessageHandler{
		TelegramClient: c.TelegramBot,
		GptClient:      c.GptClient,
	}
}
