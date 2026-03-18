package handler

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type ImageHandler struct {
	Deps *commands.Deps
}

func (i *ImageHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	image := update.Message.Photo[len(update.Message.Photo)-1]
	fileId := image.FileID

	file, err := i.Deps.Bot.GetFile(fileId)
	i.Deps.ErrorLog.LogError(err)

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", i.Deps.Bot.Token, file.FilePath)
	i.Deps.Log.Logf("Image URL: %s", url)

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
	responsePayload, err := i.Deps.GptClient.CallGPT(messages, gpt.VisionTierID, 0.8)
	i.Deps.ErrorLog.LogError(err)

	if err == nil && responsePayload != nil && len(responsePayload.Choices) > 0 {
		i.Deps.Log.Log(fmt.Sprint(responsePayload))
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	i.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
	return nil
}
