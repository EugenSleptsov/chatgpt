package storage

import "GPTBot/api/gpt"

// ToGPTMessages converts conversation history entries into a flat slice
// of gpt.Message suitable for the GPT API.
func ToGPTMessages(entries []*ConversationEntry) []gpt.Message {
	var messages []gpt.Message
	for _, entry := range entries {
		messages = append(messages, gpt.Message{Role: entry.Prompt.Role, Content: entry.Prompt.Content})
		if entry.Response != (Message{}) {
			messages = append(messages, gpt.Message{Role: entry.Response.Role, Content: entry.Response.Content})
		}
	}
	return messages
}
