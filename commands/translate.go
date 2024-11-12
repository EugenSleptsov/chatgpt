package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"fmt"
	"strings"
)

type CommandTranslate struct {
	TelegramBot *telegram.Bot
	GptClient   *gpt.GPTClient
}

const RUSSIAN = "ru"
const ENGLISH = "en"
const TURKISH = "tr"

func (c *CommandTranslate) Name() string {
	return "translate"
}

func (c *CommandTranslate) Description() string {
	return "Переводит <text> на любом языке на <lang> (по умолчанию en). Использование: /translate <lang> <text>. Доступные языки: ru, en"
}

func (c *CommandTranslate) IsAdmin() bool {
	return false
}

func (c *CommandTranslate) Execute(update telegram.Update, chat *storage.Chat) {
	if len(update.Message.CommandArguments()) == 0 {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо перевести. Использование: /translate <text>")
	} else {
		language := ENGLISH
		prompt := update.Message.CommandArguments()
		arguments := strings.Split(update.Message.CommandArguments(), " ")
		if len(arguments) > 1 {
			switch arguments[0] {
			case RUSSIAN, ENGLISH, TURKISH:
				language = arguments[0]
				prompt = strings.Join(arguments[1:], " ")
			default:
				prompt = strings.Join(arguments, " ")
			}
		}

		translationPrompt := fmt.Sprintf("Translate the following text to English: \"%s\". You should answer only with translated text without explanations and quotation marks", prompt)
		switch language {
		case RUSSIAN:
			translationPrompt = fmt.Sprintf("Переведи следующий текст на русский язык: \"%s\". Ты должен ответить только переведенным текстом без объяснений и кавычек", prompt)
		case TURKISH:
			translationPrompt = fmt.Sprintf("Türkçe'ye aşağıdaki metni çevir: \"%s\". Sade yalnızca çevrilmiş metinle cevap vermelisin, açıklamalar ve alıntı işaretleri olmadan", prompt)
		}

		systemPrompt := "You are a helpful assistant that translates."
		gptText(c.TelegramBot, chat, update.Message.MessageID, c.GptClient, systemPrompt, translationPrompt)
	}
}
