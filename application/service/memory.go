package service

import (
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
)

// Persistent memory helpers. Memory belongs to the chat — it is shared across
// all sessions. Plain functions over the domain type; no state to hold.

// AddMemory appends a new fact to the chat memory.
func AddMemory(chat *chatdomain.Chat, fact string) {
	chat.Memory = append(chat.Memory, fact)
	log.Printf("[Memory] added: %s (total: %d facts)", fact, len(chat.Memory))
}

// ClearMemory removes all facts from the chat memory. Returns the number removed.
func ClearMemory(chat *chatdomain.Chat) int {
	count := len(chat.Memory)
	chat.Memory = nil
	return count
}

// MemoryPrompt returns the memory section for the system prompt.
// Returns an empty string if no facts are stored.
func MemoryPrompt(chat *chatdomain.Chat) string {
	if len(chat.Memory) == 0 {
		return ""
	}
	return "Memory about this chat:\n- " + strings.Join(chat.Memory, "\n- ")
}

// FormatMemory returns a human-readable list of facts for the user.
func FormatMemory(chat *chatdomain.Chat) string {
	if len(chat.Memory) == 0 {
		return "Память пуста. Бот запоминает факты о вас автоматически в ходе общения."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Память бота (%d фактов):\n\n", len(chat.Memory)))
	for i, fact := range chat.Memory {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, fact))
	}
	sb.WriteString("\nДля очистки: /memory clear")
	return sb.String()
}
