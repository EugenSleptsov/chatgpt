package storage

import (
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
)

// ToGPTMessages delegates to chat.ToGPTMessages.
// Deprecated: use chat.ToGPTMessages directly.
func ToGPTMessages(entries []*chat.ConversationEntry) []ai.Message {
	return chat.ToGPTMessages(entries)
}
