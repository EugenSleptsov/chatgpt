package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/infrastructure/util"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

type CommandTranslate struct {
	Commands *service.GPTService
	Notifier *service.Notifier
}

const (
	RUSSIAN = "ru"
	ENGLISH = "en"
	TURKISH = "tr"
	GERMAN  = "de"
	FRENCH  = "fr"
)

var SupportedLanguages = map[string]string{
	ENGLISH: "Translate the following text to English",
	RUSSIAN: "Переведи следующий текст на русский язык",
	TURKISH: "Türkçe'ye aşağıdaki metni çevir",
	GERMAN:  "Übersetzen Sie den folgenden Text ins Deutsche",
	FRENCH:  "Traduisez le texte suivant en français",
}

func (c *CommandTranslate) Name() string {
	return "translate"
}

func (c *CommandTranslate) Description() string {
	return "Переводит <text> на любом языке на <lang> (по умолчанию en). Использование: /translate <lang> <text>. Доступные языки: " + strings.Join(util.MapKeys(SupportedLanguages), ", ")
}

func (c *CommandTranslate) IsAdmin() bool {
	return false
}

func (c *CommandTranslate) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	args := strings.Fields(ctx.CommandArgs)

	if len(args) == 0 {
		return reply("Пожалуйста укажите текст, который необходимо перевести. Использование: /translate <text>")
	}

	language, textToTranslate := c.extractLanguageAndText(args)
	translationPrompt := c.buildTranslationPrompt(language, textToTranslate)

	systemPrompt := "You are a helpful assistant that translates. You should answer only with translated text without explanations and quotation marks."
	return gptText(c.Commands, c.Notifier, chat, systemPrompt, translationPrompt)
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
