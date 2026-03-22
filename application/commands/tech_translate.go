package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

const TechFields = "Физика твердого тела, Полупроводниковая электроника, Полупроводниковые детекторы излучений"
const AdditionalPrompt = `
- "монокристаллическая пластина синтетического HPHT алмаза" - "single-crystal HPHT diamond plates"
`

type CommandTechTranslate struct {
	Commands *service.GPTCommandService
	Notifier *service.Notifier
}

func (c *CommandTechTranslate) Name() string {
	return "tech_translate"
}

func (c *CommandTechTranslate) Description() string {
	return "Переводит текст на технический английский язык. Использование: /tech_translate <text>"
}

func (c *CommandTechTranslate) IsAdmin() bool {
	return false
}

func (c *CommandTechTranslate) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	args := strings.Fields(ctx.CommandArgs)

	if len(args) == 0 {
		return reply("Пожалуйста укажите текст, который необходимо перевести. Использование: /tech_translate <text>")
	}

	translationPrompt := strings.Join(args, " ")

	systemPrompt := fmt.Sprintf("Ты - помощник, который переводит текст на технический английский язык. Техническая область: %s. Ты должен отвечать только переведенным текстом без объяснений и кавычек. Используй следующие соответствия, если сомневаешься в терминологии: %s", TechFields, AdditionalPrompt)
	return gptText(c.Commands, c.Notifier, chat, systemPrompt, translationPrompt)
}
