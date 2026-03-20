package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
)

type ImageHandler struct {
	Deps *commands.Deps
}

func (i *ImageHandler) Handle(update telegram.Update, chat *storage.Chat) error {
	image := update.Message.Photo[len(update.Message.Photo)-1]

	file, err := i.Deps.Bot.GetFile(image.FileID)
	if err != nil {
		i.Deps.Notifier.LogError(err)
		return nil
	}

	imageURL := i.Deps.Bot.FileURL(file.FilePath)
	i.Deps.Notifier.Logf("Image URL: %s", imageURL)

	prompt := "Пожалуйста опишите изображение"
	if update.Message.Caption != "" {
		prompt = update.Message.Caption
	}

	response, err := i.Deps.GPTService.AnalyzeImage(imageURL, prompt)
	if err != nil {
		i.Deps.Notifier.LogError(err)
		response = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	}

	i.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
	return nil
}
