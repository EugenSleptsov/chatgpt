package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/handler"
	"GPTBot/service"
	"GPTBot/storage"
	"fmt"
	"strings"
)

// Deps holds shared dependencies for all commands and handlers.
type Deps struct {
	Bot        telegram.BotAPI
	Config     *conf.Config
	ConfigPath string
	Registry   CommandRegistry
	GPTService *service.GPTService
	Notifier   *service.Notifier
	Auth       *service.Auth
}

type Command interface {
	Name() string
	Description() string
	IsAdmin() bool
	Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response
}

// reply is a shorthand for building a single text response.
func reply(text string) []handler.Response {
	return []handler.Response{{Text: text}}
}

// gptText is a convenience wrapper: calls GPTService.GPTCommand and returns the response.
func gptText(d *Deps, chat *storage.Chat, systemPrompt, userPrompt string) []handler.Response {
	session := chat.ActiveSession()
	response, err := d.GPTService.GPTCommand(session.Model, systemPrompt, userPrompt)
	if err != nil {
		d.Notifier.Logf("Error: %v", err)
		return nil
	}

	d.Notifier.Notify(fmt.Sprintf("[%s | %s]\nSystemPrompt: %s\n\nUserPrompt: %s\n\nResponse: %s", chat.Title, session.Model, systemPrompt, userPrompt, response))
	return reply(response)
}

// summarizeText reads chat log, then delegates to gptText.
func summarizeText(d *Deps, chat *storage.Chat, systemPrompt string, messageCount int) []handler.Response {
	lines, err := d.GPTService.ReadChatLog(chat.ChatID, messageCount)
	if err != nil {
		return reply("Произошла ошибка")
	}

	if len(lines) == 0 {
		return reply("История чата пуста")
	}

	chatLog := strings.Join(lines, "\n")
	return gptText(d, chat, systemPrompt, "Вот сообщения чата, которые ты должен обработать:\n\n"+chatLog)
}
