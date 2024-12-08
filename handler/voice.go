package handler

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
)

type VoiceHandler struct {
	TelegramClient *telegram.Bot
	GptClient      gpt.Client
	ErrorLogClient log.ErrorLog
}

func (v *VoiceHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	response, err := v.processAudio(update.Message.Voice.FileID)
	v.ErrorLogClient.LogError(err)
	v.TelegramClient.Reply(chat.ChatID, update.Message.MessageID, response)

	// check if message is forwarded, then we finish here
	if update.Message.ForwardFrom != nil {
		v.TelegramClient.Log(fmt.Sprintf("[%s] %s", telegram.GetChatTitle(update), "Transcribe was done"))
		return nil
	}
	update.Message.Text = response

	return nil
}

func (v *VoiceHandler) processAudio(fileID string) (string, error) {
	// Download the voice message file
	file, err := v.TelegramClient.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %w", err)
	}

	// Download the audio file content
	audioURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", v.TelegramClient.Token, file.FilePath)
	audioContent, err := util.DownloadFile(audioURL)
	if err != nil {
		return "", fmt.Errorf("error downloading audio file: %w", err)
	}

	return v.GptClient.TranscribeAudio(audioContent)
}
