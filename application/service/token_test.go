package service

import "testing"

// TestTokenUsage_LastCallInputTokens verifies that lastCallInputTokens tracks
// only the most recent API call, while InputTokens stays summed for billing.
//
// Regression: the auto-compact threshold was fed the summed InputTokens, which
// inflates with every tool-loop iteration and triggered compaction prematurely
// (history "rotating" on its own).
func TestTokenUsage_LastCallInputTokens(t *testing.T) {
	var u TokenUsage

	// Simulate a turn with three API calls (initial + two tool continuations),
	// each re-sending roughly the same 5k-token context.
	u.accumulate(RawUsage{InputTokens: 5000, OutputTokens: 100, TotalTokens: 5100}, "GPT")
	u.accumulate(RawUsage{InputTokens: 5200, OutputTokens: 80, TotalTokens: 5280}, "Continue (web_search)")
	u.accumulate(RawUsage{InputTokens: 5400, OutputTokens: 60, TotalTokens: 5460}, "Continue (generate_voice)")

	// Billing total: summed across the whole turn.
	if want := 5000 + 5200 + 5400; u.InputTokens != want {
		t.Errorf("InputTokens (billing sum) = %d, want %d", u.InputTokens, want)
	}

	// Compact threshold input: only the last call (real current context size).
	if u.lastCallInputTokens != 5400 {
		t.Errorf("lastCallInputTokens = %d, want 5400 (last call only)", u.lastCallInputTokens)
	}
}
