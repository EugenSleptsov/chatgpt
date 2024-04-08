package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strings"
	"time"
)

type CommandImagine struct{}

func (c *CommandImagine) Name() string {
	return "imagine"
}

func (c *CommandImagine) Description() string {
	return "Генерирует изображение по описанию <text> размера 1024x1024. Использование: /imagine <text>"
}

func (c *CommandImagine) IsAdmin() bool {
	return false
}

func (c *CommandImagine) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	now := time.Now()
	nextTime := chat.ImageGenNextTime
	if nextTime.After(now) && update.Message.From.ID != bot.AdminId && !util.IsIdInList(update.Message.From.ID, bot.Config.IgnoreReportIds) {
		nextTimeStr := nextTime.Format("15:04:05")
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Your next image generation will be available at %s.", nextTimeStr))
		return
	}

	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, по которому необходимо сгенерировать изображение. Использование: /imagine <text>")
	} else {
		chat.ImageGenNextTime = now.Add(time.Second * 900)
		bot.Log(fmt.Sprintf("[%s] Image prompt: \"%s\"", chat.Title, update.Message.CommandArguments()))
		err := gptImage(bot, chat.ChatID, gptClient, update.Message.CommandArguments())
		if err != nil {
			bot.Reply(chat.ChatID, update.Message.MessageID, "Произошла ошибка при генерации изображения, попробуйте позже.")
		}
	}
}

func gptImage(bot *telegram.Bot, chatID int64, gptClient *gpt.GPTClient, prompt string) error {
	imageUrl, err := gptClient.GenerateImage(prompt, gpt.ImageSize1024)
	if err != nil {
		bot.Log(fmt.Sprintf("[%d] Error generating image: %v", chatID, err))
		return err
	}

	enhancedCaption := prompt
	responsePayload, err := gptClient.CallGPT([]gpt.Message{
		{Role: "system", Content: "You are an assistant that generates natural language description (prompt) for an artificial intelligence (AI) that generates images"},
		{Role: "user", Content: fmt.Sprintf("Please improve this prompt: \"%s\". Answer with improved prompt only. Keep prompt at most 200 characters long. Your prompt must be in one sentence.", prompt)},
	}, "gpt-3.5-turbo", 0.7)
	if err == nil {
		enhancedCaption = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	err = bot.SendImage(chatID, imageUrl, enhancedCaption)
	if err != nil {
		bot.Log(fmt.Sprintf("[%d] Error sending image: %v", chatID, err))
		return err
	}

	return nil
}
