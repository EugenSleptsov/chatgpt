package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"time"
)

type CommandImagine struct {
	*Deps
}

func (c *CommandImagine) Name() string {
	return "imagine"
}

func (c *CommandImagine) Description() string {
	return "Генерирует изображение по описанию <text> размера 1024x1024. Использование: /imagine <text>"
}

func (c *CommandImagine) IsAdmin() bool {
	return false
}

func (c *CommandImagine) Execute(update telegram.Update, chat *storage.Chat) {
	now := time.Now()
	nextTime := chat.ImageGenNextTime
	if nextTime.After(now) && update.Message.From.ID != c.Bot.AdminId && !util.IsIdInList(update.Message.From.ID, c.Bot.Config.IgnoreReportIds) {
		nextTimeStr := nextTime.Format("15:04:05")
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Your next image generation will be available at %s.", nextTimeStr))
		return
	}

	if len(update.Message.CommandArguments()) == 0 {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, по которому необходимо сгенерировать изображение. Использование: /imagine <text>")
		return
	}

	chat.ImageGenNextTime = now.Add(time.Second * 900)
	aiModel := gpt.ImageEnhanceTierID
	prompt := update.Message.CommandArguments()

	c.Bot.Log(fmt.Sprintf("[%s | %s (%s)] Image prompt: \"%s\"", chat.Title, aiModel, gpt.ResolveAPIName(aiModel), prompt))

	imageURL, caption, err := c.ChatService.GenerateImage(aiModel, prompt)
	if err != nil {
		c.Bot.Log(fmt.Sprintf("[%d] Error generating image: %v", chat.ChatID, err))
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Произошла ошибка при генерации изображения, попробуйте позже.")
		return
	}

	if err := c.Bot.SendImage(chat.ChatID, imageURL, caption); err != nil {
		c.Bot.Log(fmt.Sprintf("[%d] Error sending image: %v", chat.ChatID, err))
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Произошла ошибка при генерации изображения, попробуйте позже.")
	}
}
