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
	GptClient      gpt.Client
	LogClient      log.Log
	ErrorLogClient log.ErrorLog
}

func NewUpdateHandlerFactory(telegramBot *telegram.Bot, commandFactory commands.CommandFactory, gptClient gpt.Client, logClient log.Log, errorLogClient log.ErrorLog) *ConcreteUpdateHandlerFactory {
	return &ConcreteUpdateHandlerFactory{
		TelegramBot:    telegramBot,
		CommandFactory: commandFactory,
		GptClient:      gptClient,
		LogClient:      logClient,
		ErrorLogClient: errorLogClient,
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
			ErrorLogClient: c.ErrorLogClient,
		}
	}

	if len(update.Message.Photo) > 0 {
		return &ImageHandler{
			TelegramClient: c.TelegramBot,
			GptClient:      c.GptClient,
			ErrorLogClient: c.ErrorLogClient,
			LogClient:      c.LogClient,
		}
	}

	return &MessageHandler{
		TelegramClient: c.TelegramBot,
		GptClient:      c.GptClient,
		LogClient:      c.LogClient,
		ErrorLogClient: c.ErrorLogClient,
	}
}
