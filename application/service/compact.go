package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"log"
	"strings"
)

// Auto-compact: conversation compaction.

// Compact thresholds.
const (
	compactThresholdPct = 0.75 // trigger compaction at 75% of context window
	compactKeepRecent   = 4    // always keep this many recent entries verbatim
)

// compactSystemPrompt is the summarization instruction sent to GPT.
const compactSystemPrompt = `You are a conversation summarizer. Create a detailed summary of the conversation provided.
Your summary must preserve:
1. All key facts, decisions, and context the user shared
2. The user's current request/intent
3. Any important details (names, preferences, technical specifics)
4. Memory facts that were mentioned

Be thorough but concise. The summary will replace the old messages, so nothing important should be lost.
Write the summary in the same language the conversation is in.
Do NOT use any tools. Respond with plain text only.`

// CompactService handles automatic conversation compaction.
type CompactService struct {
	GptClient       ai.Client
	CostFn          func(tierID string, inputTokens, outputTokens int) float64
	ContextWindowFn func(tierID string) int // returns max input tokens for a tier
}

// estimateTokens provides a rough token count for a string (~4 chars per token).
func estimateTokens(s string) int {
	return len(s) / 4
}

// estimateHistoryTokens sums the estimated token count of all history entries
// plus system prompt and memory.
func estimateHistoryTokens(session *chatdomain.Session, memoryPrompt string) int {
	total := estimateTokens(session.SystemPrompt) + estimateTokens(memoryPrompt)
	for _, entry := range session.History {
		total += estimateTokens(entry.Prompt.Content)
		if entry.Response != (chatdomain.Message{}) {
			total += estimateTokens(entry.Response.Content)
		}
	}
	return total
}

// ShouldCompact returns true if the session's estimated token usage exceeds
// the compaction threshold for its model.
func (cs *CompactService) ShouldCompact(session *chatdomain.Session, memoryPrompt string) bool {
	if cs.ContextWindowFn == nil {
		return false
	}
	contextWindow := cs.ContextWindowFn(session.Model)
	threshold := int(float64(contextWindow) * compactThresholdPct)
	estimated := estimateHistoryTokens(session, memoryPrompt)
	return estimated > threshold
}

// Compact summarizes the oldest history entries and replaces them with a
// single summary entry. Returns the GPT usage for the compaction call.
//
//   - It calls GPT with the old messages as context + a summarization prompt
//   - Replaces old entries with one summary entry (role: "system")
//   - Keeps the most recent entries verbatim for continuity
func (cs *CompactService) Compact(session *chatdomain.Session, memoryPrompt string) (*TokenUsage, error) {
	if len(session.History) <= compactKeepRecent {
		return nil, nil // nothing to compact
	}

	// Split: old entries to summarize | recent entries to keep
	splitIdx := len(session.History) - compactKeepRecent
	oldEntries := session.History[:splitIdx]

	// Build messages for the summarization call
	var summaryInput []ai.Message
	for _, entry := range oldEntries {
		summaryInput = append(summaryInput, ai.Message{Role: "user", Content: entry.Prompt.Content})
		if entry.Response != (chatdomain.Message{}) {
			summaryInput = append(summaryInput, ai.Message{Role: "assistant", Content: entry.Response.Content})
		}
	}

	// Add the summarization request
	summaryInput = append(summaryInput, ai.Message{
		Role:    "user",
		Content: "Please summarize the conversation above. Preserve all key facts and context.",
	})

	log.Printf("[Compact] summarizing %d entries (keeping %d recent)", splitIdx, compactKeepRecent)

	// Call GPT for summarization (no tools)
	resp, err := cs.GptClient.CallGPT(summaryInput, session.Model, compactSystemPrompt)
	if err != nil {
		log.Printf("[Compact] GPT summarization error: %v", err)
		return nil, err
	}

	summary := strings.TrimSpace(resp.OutputText())
	if summary == "" {
		log.Printf("[Compact] empty summary, skipping compaction")
		return nil, nil
	}

	// Track compaction cost
	var usage TokenUsage
	if cs.CostFn != nil {
		raw := RawUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
		if cs.CostFn != nil {
			raw.Cost = cs.CostFn(session.Model, resp.Usage.InputTokens, resp.Usage.OutputTokens)
		}
		usage.accumulate(raw, "Compact")
	}

	// Replace old entries with a single summary entry.
	summaryEntry := &chatdomain.ConversationEntry{
		Prompt: chatdomain.Message{
			Role:    "user",
			Content: "[Сжатие контекста] Саммари предыдущего разговора:\n\n" + summary,
		},
		Response: chatdomain.Message{
			Role:    "assistant",
			Content: "Понял, продолжаю с учётом контекста.",
		},
	}

	// New history: summary + recent entries
	newHistory := make([]*chatdomain.ConversationEntry, 0, 1+compactKeepRecent)
	newHistory = append(newHistory, summaryEntry)
	newHistory = append(newHistory, session.History[splitIdx:]...)
	session.History = newHistory

	log.Printf("[Compact] done: %d old entries → 1 summary + %d recent = %d total",
		splitIdx, compactKeepRecent, len(session.History))

	return &usage, nil
}
