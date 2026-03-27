package chat

import "GPTBot/domain/ai"

// ToGPTMessages converts conversation history entries into a flat slice
// of ai.Message suitable for the GPT API.
func ToGPTMessages(entries []*ConversationEntry) []ai.Message {
	var messages []ai.Message
	for _, entry := range entries {
		messages = append(messages, ai.Message{Role: entry.Prompt.Role, Content: entry.Prompt.Content})
		if entry.Response != (Message{}) {
			messages = append(messages, ai.Message{Role: entry.Response.Role, Content: entry.Response.Content})
		}
	}
	return messages
}
