package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"encoding/json"
	"fmt"
	"strings"
)

// RawUsage is a provider-agnostic snapshot of token consumption from a single
// API call. GPTService populates it from provider-specific responses;
// TokenUsage consumes it without knowing the provider.
type RawUsage struct {
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	CachedTokens    int
	ReasoningTokens int
	Cost            float64
}

// UsageStep is one line in the per-call breakdown (e.g. "GPT", "Continue (web_search)").
type UsageStep struct {
	Phase           string
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	CachedTokens    int
	ReasoningTokens int
	Cost            float64
	ToolNames       []string
}

// InputMetrics records character-level sizes of the initial request components.
// Used by admin reports to understand what contributed to input tokens.
type InputMetrics struct {
	SystemChars  int
	MemoryChars  int
	HistoryChars int
	HistoryMsgs  int
	PromptChars  int
	ToolsChars   int
}

// computeInputMetrics measures the character-level sizes of each component
// that was sent to GPT (system prompt, memory, history, last prompt, tools).
func computeInputMetrics(session *chatdomain.Session, memoryPrompt string, tools []ai.Tool) *InputMetrics {
	m := &InputMetrics{SystemChars: len(session.SystemPrompt)}
	if memoryPrompt != "" {
		m.MemoryChars = len(memoryPrompt)
	}
	history := session.History
	if len(history) > 0 {
		m.PromptChars = len(history[len(history)-1].Prompt.Content)
		history = history[:len(history)-1]
	}
	for _, entry := range history {
		m.HistoryChars += len(entry.Prompt.Content)
		m.HistoryMsgs++
		if entry.Response != (chatdomain.Message{}) {
			m.HistoryChars += len(entry.Response.Content)
			m.HistoryMsgs++
		}
	}
	if len(tools) > 0 {
		if data, err := json.Marshal(tools); err == nil {
			m.ToolsChars = len(data)
		}
	}
	return m
}

// TokenUsage tracks API token consumption and estimated cost for a request.
// Steps contains per-call breakdown; the top-level fields are grand totals.
type TokenUsage struct {
	Steps        []UsageStep   // per-call breakdown
	Input        *InputMetrics // character sizes of initial request components (nil if not measured)
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Cost         float64 // estimated cost in USD

	// lastCallInputTokens is the input_tokens of the MOST RECENT API call, not
	// the sum across the turn. The auto-compact threshold needs the real current
	// context size; summing across tool-loop iterations would inflate it and
	// trigger compaction prematurely. (Cost/InputTokens stay summed for billing.)
	lastCallInputTokens int
}

// accumulate adds the usage snapshot from a single API call to the running
// total and records a new step with the given phase label.
// toolNames lists the tools that were sent with this specific request (for reporting).
func (u *TokenUsage) accumulate(raw RawUsage, phase string, toolNames ...string) {
	step := UsageStep{
		Phase:           phase,
		InputTokens:     raw.InputTokens,
		OutputTokens:    raw.OutputTokens,
		TotalTokens:     raw.TotalTokens,
		CachedTokens:    raw.CachedTokens,
		ReasoningTokens: raw.ReasoningTokens,
		Cost:            raw.Cost,
		ToolNames:       toolNames,
	}
	u.Steps = append(u.Steps, step)
	u.InputTokens += step.InputTokens
	u.OutputTokens += step.OutputTokens
	u.TotalTokens += step.TotalTokens
	u.Cost += step.Cost
	u.lastCallInputTokens = raw.InputTokens // overwrite: tracks the latest call only
}

// addFixedCost records a fixed-price step (e.g. image generation) that has
// no token breakdown.
func (u *TokenUsage) addFixedCost(phase string, cost float64) {
	u.Steps = append(u.Steps, UsageStep{
		Phase: phase,
		Cost:  cost,
	})
	u.Cost += cost
}

// Summary returns a one-liner with grand totals, e.g. "📊 350 tok (in: 300, out: 50) · $0.0003".
func (u TokenUsage) Summary() string {
	return fmt.Sprintf("📊 %d tok (in: %d, out: %d) · $%.4f", u.TotalTokens, u.InputTokens, u.OutputTokens, u.Cost)
}

// String returns a multi-line breakdown: optional context line, one line per
// API call, and a summary.
//
// Example output (simple call):
//
//	📎 prompt: 12, system: 350, memory: 120, history: 1500 (5 msgs), tools: 800
//	  ▸ GPT [web_search, image_generation, generate_voice, update_memory]: 4536 tok (in: 4526, out: 10) · $0.0115
//	📊 4536 tok (in: 4526, out: 10) · $0.0115
func (u TokenUsage) String() string {
	if len(u.Steps) == 0 && u.Input == nil {
		return u.Summary()
	}
	var sb strings.Builder

	// Input context line — helps the admin understand what contributed to input tokens.
	if m := u.Input; m != nil {
		sb.WriteString(fmt.Sprintf("📎 prompt: %d, system: %d, memory: %d, history: %d (%d msgs), tools: %d\n",
			m.PromptChars, m.SystemChars, m.MemoryChars, m.HistoryChars, m.HistoryMsgs, m.ToolsChars))
	}

	// Per-step breakdown.
	for _, s := range u.Steps {
		if s.TotalTokens == 0 && s.Cost == 0 {
			continue
		}
		sb.WriteString("  ▸ ")
		sb.WriteString(s.Phase)

		// Show tool names if present.
		if len(s.ToolNames) > 0 {
			sb.WriteString(" [")
			sb.WriteString(strings.Join(s.ToolNames, ", "))
			sb.WriteString("]")
		}

		if s.TotalTokens > 0 {
			sb.WriteString(fmt.Sprintf(": %d tok (in: %d, out: %d", s.TotalTokens, s.InputTokens, s.OutputTokens))
			if s.CachedTokens > 0 {
				sb.WriteString(fmt.Sprintf(", cached: %d", s.CachedTokens))
			}
			if s.ReasoningTokens > 0 {
				sb.WriteString(fmt.Sprintf(", reasoning: %d", s.ReasoningTokens))
			}
			sb.WriteString(fmt.Sprintf(") · $%.4f", s.Cost))
		} else if s.Cost > 0 {
			sb.WriteString(fmt.Sprintf(": $%.4f", s.Cost))
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(u.Summary())
	return sb.String()
}
