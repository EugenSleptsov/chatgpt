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
	response, err := v.processAudio(update.Message.Voice.FileID)
	v.Deps.Notifier.LogError(err)
	v.Deps.Bot.Reply(chat.ChatID, update.Message.MessageID, response)

	// check if message is forwarded, then we finish here
	if update.Message.ForwardFrom != nil {
		v.Deps.Notifier.Notify(fmt.Sprintf("[%s] %s", telegram.GetChatTitle(update), "Transcribe was done"))
		return nil
	}
	update.Message.Text = response

	return nil
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
