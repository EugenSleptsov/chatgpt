package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/manager"
	"fmt"
)

type Worker struct {
	TelegramClient *telegram.Bot
	GptClient      *gpt.GPTClient
	ChatManager    *manager.ChatManager
	ChatLogger     *manager.ChatLogger
	CommandFactory commands.CommandFactory
	HandlerFactory handler.UpdateHandlerFactory
}

func NewWorker(telegramClient *telegram.Bot, gptClient *gpt.GPTClient, storageClient *manager.ChatManager, ChatLogger *manager.ChatLogger, commandFactory commands.CommandFactory, handlerFactory handler.UpdateHandlerFactory) *Worker {
	return &Worker{
		TelegramClient: telegramClient,
		GptClient:      gptClient,
		ChatManager:    storageClient,
		ChatLogger:     ChatLogger,
		CommandFactory: commandFactory,
		HandlerFactory: handlerFactory,
	}
}

func (w *Worker) Start(updateChan <-chan telegram.Update) {
	for update := range updateChan {
		w.ProcessUpdate(update)
		w.ChatManager.StorageClient.Save()
	}
}

func (w *Worker) ProcessUpdate(update telegram.Update) {
	// Ignore any non-Message Updates
	if update.Message == nil {
		return
	}

	chat := w.ChatManager.GetOrCreateChat(update)

	if !update.Message.IsCommand() {
		w.ChatLogger.LogMessage(update, chat)
	}

	// If no authorized users are provided, make the bot public
	if !w.TelegramClient.IsAuthorizedUser(update.Message.From.ID) {
		if update.Message.Chat.Type == "private" {
			w.TelegramClient.Reply(chat.ChatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
			w.TelegramClient.Log(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, update.Message.Text))
		}
		return
	}

	_ = w.HandlerFactory.GetHandler(update).Handle(update, chat)
}
