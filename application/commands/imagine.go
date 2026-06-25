package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"time"
)

type CommandImagine struct {
	Commands *service.GPTService
	Notifier *service.Notifier
	Auth     *service.Auth
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

func (c *CommandImagine) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	now := time.Now()
	nextTime := chat.ImageGenNextTime
	if nextTime.After(now) && !c.Auth.IsAdmin(ctx.SenderID) {
		nextTimeStr := nextTime.Format("15:04:05")
		return reply(fmt.Sprintf("Следующая генерация изображения будет доступна в %s.", nextTimeStr))
	}

	if len(ctx.CommandArgs) == 0 {
		return reply("Пожалуйста укажите текст, по которому необходимо сгенерировать изображение. Использование: /imagine <text>")
	}

	chat.ImageGenNextTime = now.Add(time.Second * 900)
	aiModel := ai.ImageEnhanceTierID
	prompt := ctx.CommandArgs

	imageURL, caption, usage, err := c.Commands.GenerateImage(aiModel, prompt)
	if err != nil {
		c.Notifier.Notify(fmt.Sprintf("[%d] Error generating image: %v", chat.ChatID, err))
		return reply("Произошла ошибка при генерации изображения, попробуйте позже.")
	}

	c.Notifier.Notify(fmt.Sprintf("[%s | %s] Image prompt: \"%s\"\n%s", chat.Title, aiModel, prompt, usage))

	return []sender.Response{{ImageURL: imageURL, Caption: caption}}
}
