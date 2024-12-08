package main

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/manager"
	"GPTBot/storage"
	"fmt"
)

type Worker struct {
	TelegramClient *telegram.Bot
	ChatManager    manager.ChatManager
	CommandFactory commands.CommandFactory
	HandlerFactory handler.UpdateHandlerFactory
}

func NewWorker(telegramClient *telegram.Bot, chatManager manager.ChatManager, commandFactory commands.CommandFactory, handlerFactory handler.UpdateHandlerFactory) *Worker {
	return &Worker{
		TelegramClient: telegramClient,
		ChatManager:    chatManager,
		CommandFactory: commandFactory,
		HandlerFactory: handlerFactory,
	}
}

func (w *Worker) Start(updateChan <-chan telegram.Update) {
	for update := range updateChan {
		w.ProcessUpdate(update)
		w.ChatManager.GetStorageClient().Save()
	}
}

func (w *Worker) ProcessUpdate(update telegram.Update) {
	if !w.isMessage(update) {
		return
	}

	chat := w.ChatManager.GetOrCreateChat(update)
	w.logIfNonCommandMessage(update, chat)

	if !w.isAuthorized(update) {
		w.handleUnauthorizedAccess(update, chat)
		return
	}

	w.handleUpdate(update, chat)
}

func (w *Worker) isMessage(update telegram.Update) bool {
	return update.Message != nil
}

func (w *Worker) logIfNonCommandMessage(update telegram.Update, chat *storage.Chat) {
	if !update.Message.IsCommand() {
		w.ChatManager.LogMessage(update, chat)
	}
}

func (w *Worker) isAuthorized(update telegram.Update) bool {
	return w.TelegramClient.IsAuthorizedUser(update.Message.From.ID)
}

func (w *Worker) handleUnauthorizedAccess(update telegram.Update, chat *storage.Chat) {
	if update.Message.Chat.Type != "private" {
		return
	}

	w.TelegramClient.Reply(chat.ChatID, update.Message.MessageID, "Sorry, you do not have access to this bot.")
	w.TelegramClient.Log(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, update.Message.Text))
}

func (w *Worker) handleUpdate(update telegram.Update, chat *storage.Chat) {
	if err := w.HandlerFactory.GetHandler(update).Handle(update, chat); err != nil {
		w.TelegramClient.Log(fmt.Sprintf("Error handling input: %v", err))
	}
}
