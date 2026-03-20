package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/service"
	"GPTBot/storage"
	"fmt"
	"strings"
)

// Deps holds shared dependencies for all commands and handlers.
type Deps struct {
	Bot        *telegram.Bot
	Config     *conf.Config
	Registry   CommandRegistry
	GPTService *service.GPTService
	Notifier   *service.Notifier
	Auth       *service.Auth
}

type Command interface {
	Name() string
	Description() string
	IsAdmin() bool
	Execute(update telegram.Update, chat *storage.Chat)
}

// gptText is a convenience wrapper: calls GPTService.GPTCommand, logs and replies.
func (d *Deps) gptText(chat *storage.Chat, messageID int, systemPrompt, userPrompt string) {
	session := chat.ActiveSession()
	response, err := d.GPTService.GPTCommand(session.Model, systemPrompt, userPrompt)
	if err != nil {
		d.Notifier.Logf("Error: %v", err)
		return
	}

	d.Notifier.Notify(fmt.Sprintf("[%s | %s]\nSystemPrompt: %s\n\nUserPrompt: %s\n\nResponse: %s", chat.Title, session.Model, systemPrompt, userPrompt, response))
	d.Bot.Reply(chat.ChatID, messageID, response)
}

// summarizeText reads chat log, then delegates to gptText.
func (d *Deps) summarizeText(chat *storage.Chat, messageID int, systemPrompt string, messageCount int) {
	lines, err := d.GPTService.ReadChatLog(chat.ChatID, messageCount)
	if err != nil {
		d.Bot.Reply(chat.ChatID, messageID, "Произошла ошибка")
		return
	}

	if len(lines) == 0 {
		d.Bot.Reply(chat.ChatID, messageID, "История чата пуста")
		return
	}

	d.Bot.Reply(chat.ChatID, messageID, fmt.Sprintf("Обработка %d сообщений...", len(lines)))
	chatLog := strings.Join(lines, "\n")
	d.gptText(chat, messageID, systemPrompt, "Вот сообщения чата, которые ты должен обработать:\n\n"+chatLog)
}
