package handler

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type ImageHandler struct {
	TelegramClient *telegram.Bot
	GptClient      gpt.Client
	ErrorLogClient log.ErrorLog
	LogClient      log.Log
}

func (i *ImageHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	image := update.Message.Photo[len(update.Message.Photo)-1]
	fileId := image.FileID

	file, err := i.TelegramClient.GetFile(fileId)
	i.ErrorLogClient.LogError(err)

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", i.TelegramClient.Token, file.FilePath)
	i.LogClient.Logf("Image URL: %s", url)

	prompt := "Пожалуйста опишите изображение"
	if update.Message.Caption != "" {
		prompt = update.Message.Caption
	}

	messages := []gpt.Message{
		{Role: "user", Content: []gpt.Content{
			{Type: gpt.TypeText, Text: prompt},
			{Type: gpt.TypeImageUrl, ImageUrl: gpt.ImageUrl{Url: url}},
		}},
	}

	response := "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	responsePayload, err := i.GptClient.CallGPT(messages, gpt.ModelGPT4Vision, 0.8)
	i.ErrorLogClient.LogError(err)

	if len(responsePayload.Choices) > 0 {
		i.LogClient.Log(fmt.Sprint(responsePayload))
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	i.TelegramClient.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
	return nil
}
