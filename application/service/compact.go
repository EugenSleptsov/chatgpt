package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"fmt"
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
	CostFn          CostFunc
	ContextWindowFn func(tierID string) int // returns max input tokens for a tier

	// Archive is the supermemory raw transcript store; snapshots summarize
	// its uncovered tail into memory nodes (may be nil — snapshots disabled).
	Archive chatdomain.Archive

	// Circuit breaker state (per-process, not persisted).
	// Claude Code tracks this in AutoCompactTrackingState.
	consecutiveFailures int
	// Separate breakers for the supermemory operations so their failures do
	// not block regular history compaction (and vice versa).
	metaFailures int
	snapFailures int
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
//
// Compaction does not create memory nodes — with Supermemory enabled the
// evicted entries are already in the raw archive, and the snapshot mechanism
// (Snapshot) covers archived lines into nodes independently of eviction.
func (cs *CompactService) Compact(chat *chatdomain.Chat, session *chatdomain.Session, memoryPrompt string) (*TokenUsage, error) {
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

	// The summary entry inherits the evicted entries' archive range so
	// ArchiveHistory never re-archives the synthetic entry.
	srcFrom, srcTo := archiveRangeOfEntries(oldEntries)

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
		ArchiveFrom: srcFrom,
		ArchiveTo:   srcTo,
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

// --- Supermemory snapshots ---

// snapshotThresholdTokens is how much un-summarized archived conversation
// accumulates before a snapshot node is created. Deliberately low compared to
// the context-eviction threshold: nodes should appear while the conversation
// is still in context, so memory visibly fills up as you chat.
const snapshotThresholdTokens = 3000

// snapshotPrompt summarizes a raw transcript chunk into a memory node,
// hook-first like metaCompactPrompt.
const snapshotPrompt = `You are writing a long-term memory entry from a chunk of chat transcript.
Preserve all key facts, decisions, names, numbers and the user's intents. The raw transcript stays retrievable, so summarize faithfully rather than exhaustively.
Write in the same language the conversation is in.
Format: the FIRST line of your reply must be a short hook — one line (max 80 chars) naming what this chunk was about, like a headline. Then an empty line, then the summary.
Do NOT use any tools. Respond with plain text only.`

// Snapshot summarizes the archive's uncovered tail [chat.ArchiveSnapshotTo,
// end) into one memory node and advances the pointer. With force=false it
// waits until the tail exceeds snapshotThresholdTokens; force=true snapshots
// whatever is there (used before /clear and session deletion so nothing is
// ever lost). No-op (nil, nil) when the feature is off, the tail is trivial,
// or the breaker is tripped.
func (cs *CompactService) Snapshot(chat *chatdomain.Chat, model string, force bool) (*TokenUsage, error) {
	if chat == nil || !chat.Settings.Supermemory || cs.Archive == nil {
		return nil, nil
	}
	if cs.snapFailures >= maxConsecutiveCompactFailures {
		return nil, nil
	}
	end, err := cs.Archive.Count(chat.ChatID)
	if err != nil {
		log.Printf("[Snapshot] archive count error: %v", err)
		return nil, err
	}
	from := chat.ArchiveSnapshotTo
	if end-from < 2 {
		return nil, nil // nothing or a lone line — not worth a node
	}
	msgs, err := cs.Archive.ReadRange(chat.ChatID, from, end)
	if err != nil {
		log.Printf("[Snapshot] archive read error: %v", err)
		return nil, err
	}

	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}
	transcript := sb.String()
	if !force && estimateTokens(transcript) < snapshotThresholdTokens {
		return nil, nil
	}

	resp, err := cs.GptClient.CallGPT([]ai.Message{{Role: "user", Content: transcript}}, model, snapshotPrompt)
	if err != nil {
		log.Printf("[Snapshot] GPT error: %v", err)
		cs.snapFailures++
		return nil, err
	}
	text := strings.TrimSpace(resp.OutputText())
	if text == "" {
		log.Printf("[Snapshot] empty summary, skipping")
		cs.snapFailures++
		return nil, nil
	}
	cs.snapFailures = 0

	var usage TokenUsage
	usage.add(extractUsage(resp, model, "Snapshot", cs.CostFn))

	hook, body := splitHookSummary(text)
	node := chat.AddMemoryNode(hook, body, from, end, 0)
	chat.ArchiveSnapshotTo = end
	log.Printf("[Snapshot] node #%d covers archive lines [%d,%d): %q", node.ID, from, end, hook)
	return &usage, nil
}

// --- Supermemory meta-compaction ---

// metaCompactBatch is how many of the oldest root nodes one meta-compaction
// folds into a single parent node. LSM semantics: the coldest block is merged
// and shifted down; clustering is deterministic (oldest first) rather than
// model-chosen, which keeps the operation predictable — search still reaches
// the children directly.
const metaCompactBatch = 10

// metaCompactPrompt asks the model to merge several node summaries into one
// parent summary, hook-first like compactSystemPrompt.
const metaCompactPrompt = `You are merging several long-term memory summaries into one parent summary.
Preserve every distinct fact, decision and detail — the child summaries stay readable below this parent, so be a faithful table of contents rather than a replacement: name each covered topic explicitly.
Write in the same language the summaries are in.
Format: the FIRST line of your reply must be a short hook — one line (max 80 chars) naming what this group of memories covers. Then an empty line, then the merged summary.
Do NOT use any tools. Respond with plain text only.`

// MetaCompact folds the oldest non-pinned root nodes into one parent node when
// the supermemory index outgrows its prompt budget. Children are linked, never
// rewritten — the operation is lossless and reversible. No-op (nil, nil) when
// the feature is off, the index fits, or the breaker is tripped.
func (cs *CompactService) MetaCompact(chat *chatdomain.Chat, model string) (*TokenUsage, error) {
	if chat == nil || !chat.Settings.Supermemory {
		return nil, nil
	}
	if cs.metaFailures >= maxConsecutiveCompactFailures {
		return nil, nil
	}
	roots := chat.RootMemoryNodes()
	if len(roots) <= supermemoryIndexLimit {
		return nil, nil
	}

	batch := make([]*chatdomain.MemoryNode, 0, metaCompactBatch)
	for _, n := range roots {
		if n.Pinned {
			continue
		}
		batch = append(batch, n)
		if len(batch) == metaCompactBatch {
			break
		}
	}
	if len(batch) < 2 {
		return nil, nil // everything pinned — nothing to fold
	}

	var sb strings.Builder
	sb.WriteString("Merge these memory nodes:\n\n")
	for _, n := range batch {
		sb.WriteString(fmt.Sprintf("#%d — %s\n%s\n\n", n.ID, n.Hook, n.Summary))
	}
	resp, err := cs.GptClient.CallGPT([]ai.Message{{Role: "user", Content: sb.String()}}, model, metaCompactPrompt)
	if err != nil {
		log.Printf("[MetaCompact] GPT error: %v", err)
		cs.metaFailures++
		return nil, err
	}
	merged := strings.TrimSpace(resp.OutputText())
	if merged == "" {
		log.Printf("[MetaCompact] empty merge result, skipping")
		cs.metaFailures++
		return nil, nil
	}
	cs.metaFailures = 0

	var usage TokenUsage
	usage.add(extractUsage(resp, model, "MetaCompact", cs.CostFn))

	hook, body := splitHookSummary(merged)
	parent := chat.AddMemoryNode(hook, body, 0, 0, 0)
	parent.Children = make([]int, 0, len(batch))
	for _, n := range batch {
		parent.Children = append(parent.Children, n.ID)
	}
	log.Printf("[MetaCompact] node #%d folds %d children: %q", parent.ID, len(batch), hook)
	return &usage, nil
}
