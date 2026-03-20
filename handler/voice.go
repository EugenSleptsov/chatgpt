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

func (v *VoiceHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	transcription, err := v.processAudio(update.Message.Voice.FileID)
	if err != nil {
		v.Deps.Notifier.LogError(err)
		v.Deps.Bot.Reply(chat.ChatID, update.Message.MessageID, "Не удалось обработать голосовое сообщение.")
		return nil
	}

	// Echo transcription so user sees what was heard
	v.Deps.Bot.Reply(chat.ChatID, update.Message.MessageID, transcription)

	// Forwarded voice: transcribe only, no GPT
	if update.Message.ForwardFrom != nil {
		v.Deps.Notifier.Notify(fmt.Sprintf("[%s] Transcribe was done", telegram.GetChatTitle(update)))
		return nil
	}

	// GPT response to the transcribed text
	response, err := v.Deps.GPTService.ChatCompletion(chat, transcription)
	v.Deps.Notifier.LogError(err)

	// Text reply
	v.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)

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
