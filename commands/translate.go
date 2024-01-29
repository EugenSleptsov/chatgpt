package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type CommandTranslate struct{}

const RUSSIAN = "ru"
const ENGLISH = "en"

func (c *CommandTranslate) Name() string {
	return "translate"
}

func (c *CommandTranslate) Description() string {
	return "Переводит <text> на любом языке на <lang> (по умолчанию en). Использование: /translate <lang> <text>. Доступные языки: ru, en"
}

func (c *CommandTranslate) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо перевести. Использование: /translate <text>")
	} else {
		language := ENGLISH
		prompt := update.Message.CommandArguments()
		arguments := strings.Split(update.Message.CommandArguments(), " ")
		if len(arguments) > 1 {
			if arguments[0] == RUSSIAN {
				language = RUSSIAN
				prompt = strings.Join(arguments[1:], " ")
			}
		}

		translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks", prompt)
		switch language {
		case RUSSIAN:
			translationPrompt = fmt.Sprintf("Переведи следующий текст на русский язык: \"%s\". Ты должен ответить только переведенным текстом без объяснений и кавычек", prompt)
		}

		systemPrompt := "You are a helpful assistant that translates."
		gptText(bot, chat, update.Message.MessageID, gptClient, systemPrompt, translationPrompt)
	}
}
