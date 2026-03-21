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

func (v *VoiceHandler) Handle(ctx *telegram.UpdateContext, chat *storage.Chat) error {
	transcription, err := v.processAudio(ctx.Msg.Voice.FileID)
	if err != nil {
		v.Deps.Notifier.LogError(err)
		v.Deps.Bot.Reply(chat.ChatID, ctx.MessageID, "Не удалось обработать голосовое сообщение.")
		return nil
	}

	// Echo transcription so user sees what was heard
	v.Deps.Bot.Reply(chat.ChatID, ctx.MessageID, transcription)

	// Forwarded voice: transcribe only, no GPT
	if ctx.Msg.ForwardFrom != nil {
		v.Deps.Notifier.Notify(fmt.Sprintf("[%s] Transcribe was done", ctx.ChatTitle()))
		return nil
	}

	// GPT response to the transcribed text
	response, err := v.Deps.GPTService.ChatCompletion(chat, transcription)
	v.Deps.Notifier.LogError(err)

	// Text reply
	v.Deps.Bot.ReplyMarkdown(chat.ChatID, ctx.MessageID, response, chat.Settings.UseMarkdown)

	// Audio reply
	audioBytes, err := v.Deps.GPTService.GenerateVoice(response)
	if err != nil {
		v.Deps.Notifier.LogError(err)
		return nil
	}
	return v.Deps.Bot.AudioUpload(chat.ChatID, audioBytes)
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
