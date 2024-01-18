package main

import (
	"GPTBot/api/gpt"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
)

type ConversationEntry struct {
	Prompt   gpt.Message
	Response gpt.Message
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
	var history []*ConversationEntry
	for _, entry := range storageHistory {
		prompt := entry.Prompt
		response := entry.Response

		history = append(history, &ConversationEntry{
			Prompt:   gpt.Message{Role: prompt.Role, Content: prompt.Content},
			Response: gpt.Message{Role: response.Role, Content: response.Content},
		})
	}

	var messages []gpt.Message
	for _, entry := range history {
		messages = append(messages, entry.Prompt)
		if entry.Response != (gpt.Message{}) {
			messages = append(messages, entry.Response)
		}
	}
	return messages
}
