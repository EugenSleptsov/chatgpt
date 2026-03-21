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
	Deps           *commands.Deps
	ChatManager    manager.ChatManager
	Router         *handler.Router
	Dispatcher     *handler.Dispatcher
	ResponseSender *handler.ResponseSender
}

func NewWorker(
	deps *commands.Deps,
	chatManager manager.ChatManager,
	router *handler.Router,
	dispatcher *handler.Dispatcher,
	sender *handler.ResponseSender,
) *Worker {
	return &Worker{
		Deps:           deps,
		ChatManager:    chatManager,
		Router:         router,
		Dispatcher:     dispatcher,
		ResponseSender: sender,
	}
}

func (w *Worker) Start(updateChan <-chan telegram.Update) {
	for update := range updateChan {
		w.ProcessUpdate(update)
		w.ChatManager.Save()
	}
}

func (w *Worker) ProcessUpdate(update telegram.Update) {
	ctx := telegram.NewUpdateContext(update)
	if ctx == nil {
		return
	}

	chat := w.ChatManager.GetOrCreateChat(ctx)
	w.ChatManager.LogMessage(ctx, chat)

	if ctx.IsGroup {
		if ctx.IsCommand && !w.Deps.Auth.IsAuthorized(ctx.SenderID) {
			return
		}
	} else {
		if !w.Deps.Auth.IsAuthorized(ctx.SenderID) {
			w.handleUnauthorizedAccess(ctx, chat)
			return
		}
	}

	// 1. Handler normalizes the update into a Request
	h := w.Router.Route(ctx)
	if h == nil {
		return
	}
	req := h.Handle(ctx, chat)
	if req == nil {
		w.ChatManager.MarkDirty(chat.ChatID)
		return // nothing to process (stickers, edits, errors)
	}

	// 2. Dispatcher routes to command or conversational branch
	responses := w.Dispatcher.Process(ctx, chat, req)

	// 3. ResponseSender delivers responses to Telegram
	w.ResponseSender.Send(chat.ChatID, ctx.MessageID, responses)
	w.ChatManager.MarkDirty(chat.ChatID)
}

func (w *Worker) handleUnauthorizedAccess(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if ctx.Msg.Chat.Type != "private" {
		return
	}
	w.Deps.Bot.Reply(chat.ChatID, ctx.MessageID, "Извините, у вас нет доступа к этому боту.")
	w.Deps.Notifier.Notify(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, ctx.Text))
}
