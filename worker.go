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
	Deps        *commands.Deps
	ChatManager manager.ChatManager
	Router      *handler.Router
}

func NewWorker(deps *commands.Deps, chatManager manager.ChatManager, router *handler.Router) *Worker {
	return &Worker{
		Deps:        deps,
		ChatManager: chatManager,
		Router:      router,
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

	// Group chats: let ALL messages through so handlers can log context.
	// Commands in groups still require authorization.
	if ctx.IsGroup {
		if ctx.IsCommand && !w.Deps.Auth.IsAuthorized(ctx.SenderID) {
			return
		}
	} else {
		// Private chats: strict authorization.
		if !w.Deps.Auth.IsAuthorized(ctx.SenderID) {
			w.handleUnauthorizedAccess(ctx, chat)
			return
		}
	}

	h := w.Router.Route(ctx)
	if h == nil {
		return
	}
	if err := h.Handle(ctx, chat); err != nil {
		w.Deps.Notifier.Notify(fmt.Sprintf("Error handling input: %v", err))
	}
	w.ChatManager.MarkDirty(chat.ChatID)
}

func (w *Worker) handleUnauthorizedAccess(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if ctx.Msg.Chat.Type != "private" {
		return
	}
	w.Deps.Bot.Reply(chat.ChatID, ctx.MessageID, "Извините, у вас нет доступа к этому боту.")
	w.Deps.Notifier.Notify(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, ctx.Text))
}
