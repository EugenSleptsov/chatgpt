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
	GptClient   gpt.Client
}

const (
	RUSSIAN = "ru"
	ENGLISH = "en"
	TURKISH = "tr"
	GERMAN  = "de"
)

var SupportedLanguages = map[string]string{
	ENGLISH: "Translate the following text to English",
	RUSSIAN: "Переведи следующий текст на русский язык",
	TURKISH: "Türkçe'ye aşağıdaki metni çevir",
	GERMAN:  "Übersetzen Sie den folgenden Text ins Deutsche",
}

func (c *CommandTranslate) Name() string {
	return "translate"
}

func (c *CommandTranslate) Description() string {
	return "Переводит <text> на любом языке на <lang> (по умолчанию en). Использование: /translate <lang> <text>. Доступные языки: " + strings.Join(arrayKeys(SupportedLanguages), ", ")
}

func (c *CommandTranslate) IsAdmin() bool {
	return false
}

func (c *CommandTranslate) Execute(update telegram.Update, chat *storage.Chat) {
	args := strings.Fields(update.Message.CommandArguments())

	if len(args) == 0 {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Пожалуйста укажите текст, который необходимо перевести. Использование: /translate <text>")
		return
	}

	language, textToTranslate := c.extractLanguageAndText(args)
	translationPrompt := c.buildTranslationPrompt(language, textToTranslate)

	systemPrompt := "You are a helpful assistant that translates. You should answer only with translated text without explanations and quotation marks."
	gptText(c.TelegramBot, chat, update.Message.MessageID, c.GptClient, systemPrompt, translationPrompt)
}

func (c *CommandTranslate) extractLanguageAndText(args []string) (string, string) {
	language := ENGLISH // Default language
	textToTranslate := strings.Join(args, " ")

	if len(args) > 1 && isSupportedLanguage(args[0]) {
		language = args[0]
		textToTranslate = strings.Join(args[1:], " ")
	}
	return language, textToTranslate
}

func (c *CommandTranslate) buildTranslationPrompt(language, text string) string {
	return fmt.Sprintf("%s: \"%s\".", SupportedLanguages[language], text)
}

func isSupportedLanguage(lang string) bool {
	_, exists := SupportedLanguages[lang]
	return exists
}

func arrayKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
