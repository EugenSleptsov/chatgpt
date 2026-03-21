package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
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

func (c *CommandImagine) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	now := time.Now()
	nextTime := chat.ImageGenNextTime
	if nextTime.After(now) && !c.Auth.IsAdmin(ctx.SenderID) {
		nextTimeStr := nextTime.Format("15:04:05")
		c.Bot.Reply(chat.ChatID, ctx.MessageID, fmt.Sprintf("Следующая генерация изображения будет доступна в %s.", nextTimeStr))
		return
	}

	if len(ctx.Msg.CommandArguments()) == 0 {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Пожалуйста укажите текст, по которому необходимо сгенерировать изображение. Использование: /imagine <text>")
		return
	}

	chat.ImageGenNextTime = now.Add(time.Second * 900)
	aiModel := gpt.ImageEnhanceTierID
	prompt := ctx.Msg.CommandArguments()

	c.Notifier.Notify(fmt.Sprintf("[%s | %s (%s)] Image prompt: \"%s\"", chat.Title, aiModel, gpt.ResolveAPIName(aiModel), prompt))

	imageURL, caption, err := c.GPTService.GenerateImage(aiModel, prompt)
	if err != nil {
		c.Notifier.Notify(fmt.Sprintf("[%d] Error generating image: %v", chat.ChatID, err))
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Произошла ошибка при генерации изображения, попробуйте позже.")
		return
	}

	if err := c.Bot.SendImage(chat.ChatID, imageURL, caption); err != nil {
		c.Notifier.Notify(fmt.Sprintf("[%d] Error sending image: %v", chat.ChatID, err))
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "Произошла ошибка при генерации изображения, попробуйте позже.")
	}
}
