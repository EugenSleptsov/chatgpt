package app

import (
	"GPTBot/api/telegram"
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/decoder"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

type Worker struct {
	Auth           *service.Auth
	Bot            sender.MessageSender
	BotUsername    string
	Notifier       *service.Notifier
	ChatService    *service.ChatService
	Decoder        *decoder.Decoder
	ResponseSender *sender.ResponseSender
}

func NewWorker(
	auth *service.Auth,
	bot sender.MessageSender,
	botUsername string,
	notifier *service.Notifier,
	chatService *service.ChatService,
	dec *decoder.Decoder,
	sender *sender.ResponseSender,
) *Worker {
	return &Worker{
		Auth:           auth,
		Bot:            bot,
		BotUsername:    botUsername,
		Notifier:       notifier,
		ChatService:    chatService,
		Decoder:        dec,
		ResponseSender: sender,
	}
}

func (w *Worker) Start(updateChan <-chan telegram.Update) {
	for update := range updateChan {
		w.ProcessUpdate(update)
		w.ChatService.Save()
	}
}

func (w *Worker) ProcessUpdate(update telegram.Update) {
	tgCtx := telegram.NewUpdateContext(update)
	if tgCtx == nil {
		return
	}

	ctx := toRequestContext(tgCtx)

	chat := w.ChatService.GetOrCreateChat(ctx)
	if !ctx.IsCallback {
		w.ChatService.LogMessage(ctx, chat)
	}

	w.consumePendingInput(ctx, chat)

	if ctx.IsGroup {
		if ctx.IsCommand && !w.Auth.IsAuthorized(ctx.SenderID) {
			return
		}
	} else {
		if !w.Auth.IsAuthorized(ctx.SenderID) {
			w.handleUnauthorizedAccess(ctx, chat)
			return
		}
	}

	// 1. Decoder picks the right executor for this update type
	exec := w.Decoder.Decode(ctx)
	if exec == nil {
		return
	}

	// 2. Executor produces responses
	responses := exec.Execute(ctx, chat)

	// 3. ResponseSender delivers responses to Telegram. Button taps edit the
	//    originating message in place; everything else is a new message.
	if ctx.IsCallback {
		w.ResponseSender.Edit(chat.ChatID, ctx.MessageID, ctx.CallbackID, responses)
	} else {
		w.ResponseSender.Send(chat.ChatID, ctx.MessageID, responses)
	}
	w.ChatService.MarkDirty(chat.ChatID)
}

// consumePendingInput implements the button → free-text input flow. When a chat
// is awaiting input (set by a command that issued a ForceReply prompt) and the
// user replies to the bot with plain text, that text is routed to the awaited
// command as its arguments. The pending flag is cleared after one cycle whether
// it was consumed or abandoned (e.g. the user ran another command instead).
func (w *Worker) consumePendingInput(ctx *pipeline.RequestContext, chat *chat.Chat) {
	if chat.PendingInput == "" {
		return
	}
	if !ctx.IsCommand && !ctx.IsCallback && ctx.Text != "" && ctx.ReplyToUsername == w.BotUsername {
		ctx.IsCommand = true
		ctx.CommandName = chat.PendingInput
		ctx.CommandArgs = ctx.Text
	}
	chat.PendingInput = ""
}

func (w *Worker) handleUnauthorizedAccess(ctx *pipeline.RequestContext, chat *chat.Chat) {
	if ctx.IsGroup {
		return
	}
	w.Bot.Reply(chat.ChatID, ctx.MessageID, "Извините, у вас нет доступа к этому боту.")
	w.Notifier.Notify(fmt.Sprintf("[%s]\nMessage: %s", chat.Title, ctx.Text))
}

// toRequestContext converts a transport-specific telegram.UpdateContext into a
// transport-agnostic pipeline.RequestContext. This is the ONLY place in the
// codebase where we bridge from Telegram types to the pipeline model.
func toRequestContext(tc *telegram.UpdateContext) *pipeline.RequestContext {
	rc := &pipeline.RequestContext{
		ChatID:     tc.ChatID,
		MessageID:  tc.MessageID,
		SenderID:   tc.SenderID,
		SenderName: tc.SenderName,
		Text:       tc.Text,
		ChatTitle:  tc.ChatTitle(),
		IsEdited:   tc.IsEdited,
		IsGroup:    tc.IsGroup,
		IsCommand:  tc.IsCommand,
		IsPhoto:    tc.IsPhoto,
		IsVoice:    tc.IsVoice,
		IsSticker:  tc.IsSticker,
	}

	// A callback (inline-button tap) is routed exactly like a typed command:
	// the button payload "<command>:<args>" becomes CommandName/CommandArgs so
	// the existing CommandExecutor handles it. Delivery differs (edit in place).
	if tc.IsCallback {
		rc.IsCallback = true
		rc.CallbackID = tc.CallbackID
		rc.IsCommand = true
		rc.CommandName, rc.CommandArgs = parseCallbackData(tc.CallbackData)
		return rc
	}

	msg := tc.Msg

	if tc.IsCommand {
		rc.CommandName = msg.Command()
		rc.CommandArgs = msg.CommandArguments()
	}

	if tc.IsPhoto && len(msg.Photo) > 0 {
		rc.PhotoFileID = msg.Photo[len(msg.Photo)-1].FileID
	}

	if tc.IsVoice && msg.Voice != nil {
		rc.VoiceFileID = msg.Voice.FileID
	}

	if tc.IsSticker && msg.Sticker != nil {
		rc.StickerEmoji = msg.Sticker.Emoji
	}

	rc.Caption = msg.Caption

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		rc.ReplyToUsername = msg.ReplyToMessage.From.UserName
	}

	rc.IsForwarded = msg.ForwardFrom != nil

	return rc
}

// parseCallbackData splits an inline-button payload "<command>:<args>" into its
// command name and argument string. Missing parts yield empty strings.
func parseCallbackData(data string) (name, args string) {
	name, args, _ = strings.Cut(data, ":")
	return name, args
}
