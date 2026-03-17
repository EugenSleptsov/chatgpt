package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"log"
	"strings"
)

// Deps holds shared dependencies for all commands.
type Deps struct {
	Bot       *telegram.Bot
	GptClient gpt.Client
	Registry  CommandRegistry
}

type Command interface {
	Name() string
	Description() string
	IsAdmin() bool
	Execute(update telegram.Update, chat *storage.Chat)
}

func (d *Deps) gptText(chat *storage.Chat, messageID int, systemPrompt, userPrompt string) {
	responsePayload, err := d.GptClient.CallGPT([]gpt.Message{
		{Role: "system", Content: []gpt.Content{{Type: gpt.TypeText, Text: systemPrompt}}},
		{Role: "user", Content: []gpt.Content{{Type: gpt.TypeText, Text: userPrompt}}},
	}, chat.Settings.Model, 0.6)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	response := "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"
	if len(responsePayload.Choices) > 0 {
		response = strings.TrimSpace(fmt.Sprintf("%v", responsePayload.Choices[0].Message.Content))
	}

	d.Bot.Log(fmt.Sprintf("[%s | %s]\nSystemPrompt: %s\n\nUserPrompt: %s\n\nResponse: %s", chat.Title, chat.Settings.Model, systemPrompt, userPrompt, response))
	d.Bot.Reply(chat.ChatID, messageID, response)
}

func (d *Deps) summarizeText(chat *storage.Chat, messageID int, systemPrompt string, messageCount int) {
	lines, err := util.ReadLastLines(fmt.Sprintf("log/%d.log", chat.ChatID), messageCount)
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
