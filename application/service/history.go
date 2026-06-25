package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"GPTBot/infrastructure/util"
	"fmt"
	"strings"
	"time"
)

// AppendHistory adds a user message to the session history.
// No trimming is done here — context management is handled entirely by
// CompactService.ShouldCompact / Compact, mirroring Claude Code's approach
// where autoCompactIfNeeded replaces any hard message limit.
func AppendHistory(session *chatdomain.Session, prompt chatdomain.Message) {
	entry := &chatdomain.ConversationEntry{Prompt: prompt}
	session.History = append(session.History, entry)
}

// AttachResponse sets the assistant response on the last history entry.
func AttachResponse(session *chatdomain.Session, response chatdomain.Message) {
	if len(session.History) == 0 {
		return
	}
	session.History[len(session.History)-1].Response = response
}

// ClearHistory removes all entries from the session history.
func ClearHistory(session *chatdomain.Session) {
	session.History = nil
}

// RollbackHistory removes the last n entries. Returns the actual number removed.
func RollbackHistory(session *chatdomain.Session, n int) int {
	if n > len(session.History) {
		n = len(session.History)
	}
	if n > 0 {
		session.History = session.History[:len(session.History)-n]
	}
	return n
}

// HistoryMessages converts the full session history to GPT API messages.
func HistoryMessages(session *chatdomain.Session) []ai.Message {
	return chatdomain.ToGPTMessages(session.History)
}

// LogGroupMessage stores a group participant's message in the active session history.
func LogGroupMessage(chat *chatdomain.Chat, author, text string) {
	session := chat.ActiveSession()
	AppendHistory(session, chatdomain.Message{
		Role:    "user",
		Content: fmt.Sprintf("%s: %s", author, text),
	})
}

// LogGroupPhoto stores a photo placeholder in the active session history.
func LogGroupPhoto(chat *chatdomain.Chat, author, description string) {
	LogGroupMessage(chat, author, fmt.Sprintf("[Фото] %s", description))
}

// LogGroupSticker stores a sticker placeholder in the active session history.
func LogGroupSticker(chat *chatdomain.Chat, author, emoji string) {
	text := "[Стикер]"
	if emoji != "" {
		text = fmt.Sprintf("[Стикер: %s]", emoji)
	}
	LogGroupMessage(chat, author, text)
}

// LogBotResponse attaches the bot's reply to the last entry of the active session.
func LogBotResponse(chat *chatdomain.Chat, text string) {
	AttachResponse(chat.ActiveSession(), chatdomain.Message{Role: "assistant", Content: text})
}

// FormatHistoryPage returns a paginated slice of formatted history chunks.
// page=1 is the latest page, page=2 is the previous, etc.
func FormatHistoryPage(session *chatdomain.Session, page, pageSize int) ([]string, int) {
	chunks := formatHistory(chatdomain.ToGPTMessages(session.History))
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

func formatHistory(history []ai.Message) []string {
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

// PromptContext holds dynamic context injected into the system prompt.
// which separates static cacheable sections from dynamic runtime context.
type PromptContext struct {
	ChatTitle   string // Telegram chat title
	IsGroup     bool   // true for group chats
	UseMarkdown bool   // whether markdown formatting is enabled
}

// BuildInstructions constructs a structured system prompt from multiple sections:
//  1. Persona/role (session system prompt)
//  2. Capabilities (available tools)
//  3. Memory (persistent facts)
//  4. Dynamic context (date/time, chat info)
//  5. Response style guidelines
func BuildInstructions(session *chatdomain.Session, memoryPrompt string, ctx *PromptContext) string {
	var parts []string

	// Section 1: Persona / Role (static, cacheable)
	if session.SystemPrompt != "" {
		parts = append(parts, session.SystemPrompt)
	}

	// Section 2: Capabilities (static, cacheable)
	parts = append(parts, `Capabilities:
- You can search the internet for up-to-date information
- You can generate images from text descriptions
- You can create voice/audio messages
- You can remember facts about the user for future conversations`)

	// Section 3: Memory (semi-static, changes infrequently)
	if memoryPrompt != "" {
		parts = append(parts, memoryPrompt)
	}

	// Section 4: Dynamic context (changes every request)
	now := time.Now()
	dynamicCtx := fmt.Sprintf("Current date and time: %s", now.Format("2006-01-02 15:04 MST"))
	if ctx != nil {
		if ctx.ChatTitle != "" {
			dynamicCtx += fmt.Sprintf("\nChat: %s", ctx.ChatTitle)
		}
		if ctx.IsGroup {
			dynamicCtx += "\nChat type: group conversation (multiple participants)"
		} else {
			dynamicCtx += "\nChat type: private conversation"
		}
	}
	parts = append(parts, dynamicCtx)

	// Section 5: Response style (static, cacheable)
	style := "Response guidelines:\n- Be concise — this is a Telegram chat, not a document.\n- Prefer short paragraphs over walls of text."
	if ctx != nil && ctx.UseMarkdown {
		style += "\n- You may use Markdown formatting (bold, italic, code blocks, lists)."
	} else if ctx != nil {
		style += "\n- Use plain text only, no Markdown."
	}
	parts = append(parts, style)

	return strings.Join(parts, "\n\n")
}
