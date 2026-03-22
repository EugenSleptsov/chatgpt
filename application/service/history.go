package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"GPTBot/infrastructure/util"
	"fmt"
	"strings"
)

// HistoryService manages conversation history within a session.
// History belongs to sessions — each session has an independent conversation thread.
type HistoryService struct{}

func NewHistoryService() *HistoryService {
	return &HistoryService{}
}

// Append adds a user message to the session history and trims to maxMessages.
func (h *HistoryService) Append(session *chatdomain.Session, prompt chatdomain.Message, maxMessages int) {
	entry := &chatdomain.ConversationEntry{Prompt: prompt}
	session.History = append(session.History, entry)
	h.Trim(session, maxMessages)
}

// AttachResponse sets the assistant response on the last history entry.
func (h *HistoryService) AttachResponse(session *chatdomain.Session, response chatdomain.Message) {
	if len(session.History) == 0 {
		return
	}
	session.History[len(session.History)-1].Response = response
}

// Trim removes the oldest entries so that at most maxMessages remain.
func (h *HistoryService) Trim(session *chatdomain.Session, maxMessages int) {
	if maxMessages > 0 && len(session.History) > maxMessages {
		session.History = session.History[len(session.History)-maxMessages:]
	}
}

// Clear removes all entries from the session history.
func (h *HistoryService) Clear(session *chatdomain.Session) {
	session.History = nil
}

// Rollback removes the last n entries. Returns the actual number removed.
func (h *HistoryService) Rollback(session *chatdomain.Session, n int) int {
	if n > len(session.History) {
		n = len(session.History)
	}
	if n > 0 {
		session.History = session.History[:len(session.History)-n]
	}
	return n
}

// Messages converts the full session history to GPT API messages.
func (h *HistoryService) Messages(session *chatdomain.Session) []ai.Message {
	return chatdomain.ToGPTMessages(session.History)
}

// LastN returns the last n history entries (or all if fewer exist).
func (h *HistoryService) LastN(session *chatdomain.Session, n int) []*chatdomain.ConversationEntry {
	history := session.History
	if len(history) > n {
		history = history[len(history)-n:]
	}
	return history
}

// LogGroupMessage stores a group participant's message in the active session history.
func (h *HistoryService) LogGroupMessage(chat *chatdomain.Chat, author, text string) {
	session := chat.ActiveSession()
	h.Append(session, chatdomain.Message{
		Role:    "user",
		Content: fmt.Sprintf("%s: %s", author, text),
	}, chat.Settings.MaxMessages)
}

// LogGroupPhoto stores a photo placeholder in the active session history.
func (h *HistoryService) LogGroupPhoto(chat *chatdomain.Chat, author, description string) {
	h.LogGroupMessage(chat, author, fmt.Sprintf("[Фото] %s", description))
}

// LogGroupSticker stores a sticker placeholder in the active session history.
func (h *HistoryService) LogGroupSticker(chat *chatdomain.Chat, author, emoji string) {
	text := "[Стикер]"
	if emoji != "" {
		text = fmt.Sprintf("[Стикер: %s]", emoji)
	}
	h.LogGroupMessage(chat, author, text)
}

// LogBotResponse attaches the bot's reply to the last entry of the active session.
func (h *HistoryService) LogBotResponse(chat *chatdomain.Chat, text string) {
	session := chat.ActiveSession()
	h.AttachResponse(session, chatdomain.Message{Role: "assistant", Content: text})
}

// FormatPage returns a paginated slice of formatted history chunks.
// page=1 is the latest page, page=2 is the previous, etc.
func (h *HistoryService) FormatPage(session *chatdomain.Session, page, pageSize int) ([]string, int) {
	chunks := h.formatHistory(chatdomain.ToGPTMessages(session.History))
	totalPages := (len(chunks) + pageSize - 1) / pageSize

	if page > totalPages {
		page = totalPages
	}

	end := len(chunks) - (page-1)*pageSize
	start := end - pageSize
	if start < 0 {
		start = 0
	}

	return chunks[start:end], totalPages
}

func (h *HistoryService) formatHistory(history []ai.Message) []string {
	if len(history) == 0 {
		return []string{"История разговоров пуста."}
	}

	var current string
	var chunks []string
	currentLen := 0

	for i, message := range history {
		line := fmt.Sprintf("%d. %s: %s\n", i+1, util.Title(message.Role), message.Content)
		lineRunes := len([]rune(line))

		if currentLen+lineRunes > 4096 {
			chunks = append(chunks, current)
			current = ""
			currentLen = 0
		}

		current += line
		currentLen += lineRunes
	}

	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}

// BuildInstructions constructs the system prompt from session settings and chat memory.
func (h *HistoryService) BuildInstructions(session *chatdomain.Session, memoryPrompt string) string {
	var parts []string
	if session.SystemPrompt != "" {
		parts = append(parts, session.SystemPrompt)
	}
	if memoryPrompt != "" {
		parts = append(parts, memoryPrompt)
	}
	return strings.Join(parts, "\n\n")
}
