package service

import (
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
)

// MemoryService manages persistent memory facts for a chat.
// Memory belongs to the chat — it is shared across all sessions.
type MemoryService struct{}

func NewMemoryService() *MemoryService {
	return &MemoryService{}
}

// Add appends a new fact to the chat memory.
func (m *MemoryService) Add(chat *chatdomain.Chat, fact string) {
	chat.Memory = append(chat.Memory, fact)
	log.Printf("[Memory] added: %s (total: %d facts)", fact, len(chat.Memory))
}

// Clear removes all facts from the chat memory. Returns the number removed.
func (m *MemoryService) Clear(chat *chatdomain.Chat) int {
	count := len(chat.Memory)
	chat.Memory = nil
	return count
}

// List returns all stored facts.
func (m *MemoryService) List(chat *chatdomain.Chat) []string {
	return chat.Memory
}

// Count returns the number of stored facts.
func (m *MemoryService) Count(chat *chatdomain.Chat) int {
	return len(chat.Memory)
}

// BuildPrompt returns the memory section for the system prompt.
// Returns an empty string if no facts are stored.
func (m *MemoryService) BuildPrompt(chat *chatdomain.Chat) string {
	if len(chat.Memory) == 0 {
		return ""
	}
	return "Memory about this chat:\n- " + strings.Join(chat.Memory, "\n- ")
}

// FormatDisplay returns a human-readable list of facts for the user.
func (m *MemoryService) FormatDisplay(chat *chatdomain.Chat) string {
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
