package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"strings"
)

const AdditionalPrompt = `
- "монокристаллическая пластина синтетического HPHT алмаза" - "single-crystal HPHT diamond plates"
`

type CommandTechTranslate struct {
	TelegramBot *telegram.Bot
	GptClient   gpt.Client
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

func (c *CommandTechTranslate) Execute(update telegram.Update, chat *storage.Chat) {
	args := strings.Fields(update.Message.CommandArguments())

	if len(args) == 0 {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо перевести. Использование: /tech_translate <text>")
		return
	}

	translationPrompt := strings.Join(args, " ")

	systemPrompt := "Ты - помощник, который переводит текст на технический английский язык. Техническая область Физика твердого тела и Полупроводниковая электроника. Ты должен отвечать только переведенным текстом без объяснений и кавычек. Используй следующие соответствия, если сомневаешься в терминологии: " + AdditionalPrompt
	gptText(c.TelegramBot, chat, update.Message.MessageID, c.GptClient, systemPrompt, translationPrompt)
}
