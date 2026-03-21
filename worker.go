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
	Pipeline    *handler.Pipeline
}

func NewWorker(deps *commands.Deps, chatManager manager.ChatManager, router *handler.Router, pipeline *handler.Pipeline) *Worker {
	return &Worker{
		Deps:        deps,
		ChatManager: chatManager,
		Router:      router,
		Pipeline:    pipeline,
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
		return // handler dealt with it internally (commands, stickers, edits)
	}

	// 2. Pipeline resolves intents and executes them
	responses := w.Pipeline.Process(ctx, chat, req)

	// 3. Deliver responses
	w.sendResponses(ctx, chat, responses)
	w.ChatManager.MarkDirty(chat.ChatID)
}

func (w *Worker) sendResponses(ctx *telegram.UpdateContext, chat *storage.Chat, responses []handler.Response) {
	for _, r := range responses {
		switch {
		case len(r.Audio) > 0:
			if err := w.Deps.Bot.AudioUpload(chat.ChatID, r.Audio); err != nil {
				w.Deps.Notifier.LogError(err)
			}
		case r.ImageURL != "":
			if err := w.Deps.Bot.SendImage(chat.ChatID, r.ImageURL, r.Caption); err != nil {
				w.Deps.Notifier.LogError(err)
			}
		case r.Text != "":
			w.Deps.Bot.ReplyMarkdown(chat.ChatID, ctx.MessageID, r.Text, r.Markdown)
		}
	}
}

func (w *Worker) handleUnauthorizedAccess(ctx *telegram.UpdateContext, chat *storage.Chat) {
	if ctx.Msg.Chat.Type != "private" {
		return
	}
	w.Deps.Bot.Reply(chat.ChatID, ctx.MessageID, "Извините, у вас нет доступа к этому боту.")
	w.Deps.Notifier.Notify(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, ctx.Text))
}
