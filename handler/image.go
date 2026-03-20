package handler

import (
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
	if chat.ChatID < 0 {
		return i.handleGroup(update, chat)
	}
	return i.handlePrivate(update, chat)
}

// handlePrivate — always analyze and respond in private chats.
func (i *ImageHandler) handlePrivate(update telegram.Update, chat *storage.Chat) error {
	image := update.Message.Photo[len(update.Message.Photo)-1]
	file, err := i.Deps.Bot.GetFile(image.FileID)
	if err != nil {
		i.Deps.Notifier.LogError(err)
		return nil
	}

	imageURL := i.Deps.Bot.FileURL(file.FilePath)
	i.Deps.Notifier.Logf("Image URL: %s", imageURL)

	prompt := "Пожалуйста, опишите изображение"
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

// handleGroup — in group chats, analyze only if bot was explicitly mentioned or replied to.
// Otherwise just log a placeholder so the conversation context stays intact.
func (i *ImageHandler) handleGroup(update telegram.Update, chat *storage.Chat) error {
	author := authorName(update)
	caption := update.Message.Caption
	botUsername := i.Deps.Bot.GetUsername()

	botMentioned := strings.Contains(caption, "@"+botUsername)
	isReplyToBot := update.Message.ReplyToMessage != nil &&
		update.Message.ReplyToMessage.From != nil &&
		update.Message.ReplyToMessage.From.UserName == botUsername

	if !botMentioned && !isReplyToBot {
		// Not addressed to bot — store placeholder, do nothing else
		i.Deps.GPTService.LogGroupPhoto(chat, author, caption)
		i.Deps.Notifier.Logf("[Group] %s → фото без упоминания, логирую как [Фото] %s", author, caption)
		return nil
	}

	// Bot was mentioned with the image — analyze it
	i.Deps.Notifier.Logf("[Group] %s → фото с упоминанием бота, анализирую", author)
	image := update.Message.Photo[len(update.Message.Photo)-1]
	file, err := i.Deps.Bot.GetFile(image.FileID)
	if err != nil {
		i.Deps.Notifier.LogError(err)
		return nil
	}

	imageURL := i.Deps.Bot.FileURL(file.FilePath)
	prompt := "Пожалуйста, опишите изображение"
	if caption != "" {
		prompt = strings.TrimSpace(strings.ReplaceAll(caption, "@"+botUsername, ""))
	}

	response, err := i.Deps.GPTService.AnalyzeImage(imageURL, prompt)
	if err != nil {
		i.Deps.Notifier.LogError(err)
		response = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	}

	// Store both the request and the response in group history
	i.Deps.GPTService.LogGroupMessage(chat, author, fmt.Sprintf("[Фото] %s", prompt))
	i.Deps.GPTService.LogBotResponse(chat, response)

	i.Deps.Bot.ReplyMarkdown(chat.ChatID, update.Message.MessageID, response, chat.Settings.UseMarkdown)
	return nil
}
