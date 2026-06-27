package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"log"
	"strings"
)

// Auto-compact: conversation compaction inspired by Claude Code CLI.
//
// Claude Code's src/services/compact/autoCompact.ts triggers compaction when
// context approaches the model's limit: threshold = contextWindow - buffer.
// It uses real API token counts (input_tokens from the last response) when
// available, and falls back to character-based estimation.
//
// Key patterns borrowed:
//   - Threshold = contextWindow - buffer (not percentage-based)
//   - Circuit breaker: stop retrying after N consecutive failures
//   - Use real API input_tokens from last response for accurate threshold check
//   - Keep recent entries verbatim for continuity

// --- Constants (modeled after Claude Code's autoCompact.ts) ---

const (
	// Buffer tokens reserved before triggering compact.
	// Claude Code uses AUTOCOMPACT_BUFFER_TOKENS = 13_000.
	// We use a larger buffer because Telegram messages are shorter and we want
	// earlier compaction to keep responses snappy.
	compactBufferTokens = 20_000

	// Always keep this many recent entries verbatim after compaction.
	compactKeepRecent = 4

	// Circuit breaker: stop auto-compact after this many consecutive failures.
	// Prevents wasting API calls when context is irrecoverably broken.
	// Claude Code uses MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES = 3.
	maxConsecutiveCompactFailures = 3
)

// compactSystemPrompt is the summarization instruction sent to GPT.
// Modeled after Claude Code's src/services/compact/prompt.ts BASE_COMPACT_PROMPT.
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

	// Circuit breaker state (per-process, not persisted).
	// Claude Code tracks this in AutoCompactTrackingState.
	consecutiveFailures int
}

// estimateTokens provides a rough token count for a string (~4 chars per token).
// Claude Code's roughTokenCountEstimation uses the same heuristic with a
// configurable bytesPerToken (default 4).
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

// getCompactThreshold returns the token count that triggers compaction.
// Formula: contextWindow - buffer (like Claude Code's getAutoCompactThreshold).
func (cs *CompactService) getCompactThreshold(model string) int {
	if cs.ContextWindowFn == nil {
		return 100_000 // safe default
	}
	contextWindow := cs.ContextWindowFn(model)
	threshold := contextWindow - compactBufferTokens
	if threshold < 10_000 {
		threshold = 10_000 // floor so we don't compact too aggressively on tiny windows
	}
	return threshold
}

// ShouldCompact returns true if the session's token usage exceeds the
// compaction threshold for its model. Uses real API token count from the
// last response when available (lastInputTokens > 0), falls back to
// character-based estimation otherwise.
//
// Claude Code's shouldAutoCompact (autoCompact.ts) uses tokenCountWithEstimation
// which prefers the last API response's usage.input_tokens over rough estimates.
func (cs *CompactService) ShouldCompact(session *chatdomain.Session, memoryPrompt string, lastInputTokens int) bool {
	// Circuit breaker check (Claude Code: MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES)
	if cs.consecutiveFailures >= maxConsecutiveCompactFailures {
		return false
	}

	threshold := cs.getCompactThreshold(session.Model)

	// Prefer real API token count over estimation
	var tokenCount int
	if lastInputTokens > 0 {
		tokenCount = lastInputTokens
	} else {
		tokenCount = estimateHistoryTokens(session, memoryPrompt)
	}

	return tokenCount > threshold
}

// Compact summarizes the oldest history entries and replaces them with a
// single summary entry. Returns the GPT usage for the compaction call.
//
// This is our Go adaptation of Claude Code's compactConversation():
//   - Calls GPT with old messages + summarization prompt
//   - Replaces old entries with one summary entry
//   - Keeps recent entries verbatim for continuity
//   - Updates circuit breaker state on success/failure
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

	log.Printf("[Compact] summarizing %d entries (keeping %d recent), threshold=%d",
		splitIdx, compactKeepRecent, cs.getCompactThreshold(session.Model))

	// Call GPT for summarization (no tools — like Claude Code's NO_TOOLS_PREAMBLE)
	resp, err := cs.GptClient.CallGPT(summaryInput, session.Model, compactSystemPrompt)
	if err != nil {
		log.Printf("[Compact] GPT summarization error: %v", err)
		cs.consecutiveFailures++
		if cs.consecutiveFailures >= maxConsecutiveCompactFailures {
			log.Printf("[Compact] circuit breaker tripped after %d consecutive failures — skipping future attempts",
				cs.consecutiveFailures)
		}
		return nil, err
	}

	summary := strings.TrimSpace(resp.OutputText())
	if summary == "" {
		log.Printf("[Compact] empty summary, skipping compaction")
		cs.consecutiveFailures++
		return nil, nil
	}

	// Success — reset circuit breaker
	cs.consecutiveFailures = 0

	// Track compaction cost
	var usage TokenUsage
	usage.add(extractUsage(resp, session.Model, "Compact", cs.CostFn))

	// Replace old entries with a single summary entry.
	// This is our equivalent of Claude Code's:
	//   this.mutableMessages.splice(0, mutableBoundaryIdx)
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
