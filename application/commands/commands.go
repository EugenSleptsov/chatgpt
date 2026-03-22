package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

type Command interface {
	Name() string
	Description() string
	IsAdmin() bool
	Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response
}

// reply is a shorthand for building a single text response.
func reply(text string) []sender.Response {
	return []sender.Response{{Text: text}}
}

// gptText is a convenience wrapper: calls GPTCommandService.GPTCommand and returns the response.
func gptText(cmds *service.GPTCommandService, notifier *service.Notifier, chat *chat.Chat, systemPrompt, userPrompt string) []sender.Response {
	session := chat.ActiveSession()
	response, usage, err := cmds.GPTCommand(session.Model, systemPrompt, userPrompt)
	if err != nil {
		notifier.Logf("Error: %v", err)
		return nil
	}

	notifier.Notify(fmt.Sprintf("[%s | %s]\nSystemPrompt: %s\n\nUserPrompt: %s\n\nResponse: %s\n\n%s", chat.Title, session.Model, systemPrompt, userPrompt, response, usage))
	return reply(response)
}

// summarizeText reads chat log, then delegates to gptText.
func summarizeText(cmds *service.GPTCommandService, chatService *service.ChatService, notifier *service.Notifier, chat *chat.Chat, systemPrompt string, messageCount int) []sender.Response {
	lines, err := chatService.ReadChatLog(chat.ChatID, messageCount)
	if err != nil {
		return reply("Произошла ошибка")
	}

	if len(lines) == 0 {
		return reply("История чата пуста")
	}

	chatLog := strings.Join(lines, "\n")
	return gptText(cmds, notifier, chat, systemPrompt, "Вот сообщения чата, которые ты должен обработать:\n\n"+chatLog)
}
