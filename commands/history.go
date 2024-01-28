package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
)

type CommandHistory struct{}

func (c *CommandHistory) Name() string {
	return "history"
}

func (c *CommandHistory) Description() string {
	return "Показывает всю сохраненную на данный момент историю разговоров в красивом форматировании."
}

func (c *CommandHistory) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	historyMessages := formatHistory(messagesFromHistory(chat.History))
	for _, message := range historyMessages {
		bot.Reply(chat.ChatID, update.Message.MessageID, message)
	}
}

func formatHistory(history []gpt.Message) []string {
	if len(history) == 0 {
		return []string{"История разговоров пуста."}
	}

	var historyMessage string
	var historyMessages []string
	characterCount := 0

	for i, message := range history {
		formattedLine := fmt.Sprintf("%d. %s: %s\n", i+1, util.Title(message.Role), message.Content)
		lineLength := len(formattedLine)

		if characterCount+lineLength > 4096 {
			historyMessages = append(historyMessages, historyMessage)
			historyMessage = ""
			characterCount = 0
		}

		historyMessage += formattedLine
		characterCount += lineLength
	}

	if len(historyMessage) > 0 {
		historyMessages = append(historyMessages, historyMessage)
	}

	return historyMessages
}

func messagesFromHistory(storageHistory []*storage.ConversationEntry) []gpt.Message {
	var messages []gpt.Message
	for _, entry := range storageHistory {
		prompt := entry.Prompt
		response := entry.Response

		messages = append(messages, gpt.Message{Role: prompt.Role, Content: prompt.Content})
		if response != (storage.Message{}) {
			messages = append(messages, gpt.Message{Role: response.Role, Content: response.Content})
		}
	}

	return messages
}
