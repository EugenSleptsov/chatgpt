package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
)

type VoiceHandler struct {
	Deps *commands.Deps
}

func (v *VoiceHandler) Match(ctx *telegram.UpdateContext) bool {
	return ctx.IsVoice && !ctx.IsEdited
}

// Handle transcribes voice into text, producing a normalized Request.
func (v *VoiceHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) *Request {
	transcription, err := v.processAudio(ctx.Msg.Voice.FileID)
	if err != nil {
		v.Deps.Notifier.LogError(err)
		v.Deps.Bot.Reply(chat.ChatID, ctx.MessageID, "Не удалось обработать голосовое сообщение.")
		return nil
	}

	return &Request{
		Text:          transcription,
		OriginalMedia: MediaVoice,
		IsForwarded:   ctx.Msg.ForwardFrom != nil,
	}
}

func (v *VoiceHandler) processAudio(fileID string) (string, error) {
	file, err := v.Deps.Bot.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %w", err)
	}

	audioContent, err := util.DownloadFile(v.Deps.Bot.FileURL(file.FilePath))
	if err != nil {
		return "", fmt.Errorf("error downloading audio file: %w", err)
	}

	return v.Deps.GPTService.TranscribeAudio(audioContent)
}
