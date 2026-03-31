package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"errors"
	"fmt"
	"testing"
)

// --- stub AI client for compact tests ---

type stubCompactClient struct {
	response *ai.Response
	err      error
	calls    int // track how many times CallGPT was invoked
}

func (s *stubCompactClient) CallGPT(_ []ai.Message, model string, _ string, _ ...ai.Tool) (*ai.Response, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.response, nil
}
func (s *stubCompactClient) ContinueWithToolOutputs(_ string, _ []ai.ToolCallOutput, _ string, _ string, _ ...ai.Tool) (*ai.Response, error) {
	return nil, nil
}
func (s *stubCompactClient) GenerateImage(_ string, _ string) (string, error) { return "", nil }
func (s *stubCompactClient) GenerateVoice(_ string, _ string, _ string) ([]byte, error) {
	return nil, nil
}
func (s *stubCompactClient) TranscribeAudio(_ []byte) (string, error) { return "", nil }

var _ ai.Client = (*stubCompactClient)(nil)

// --- helpers ---

func makeSession(n int) *chatdomain.Session {
	s := &chatdomain.Session{
		ID:           1,
		Topic:        "test",
		Model:        "basic",
		SystemPrompt: "You are a test assistant.",
		History:      make([]*chatdomain.ConversationEntry, 0, n),
	}
	for i := 0; i < n; i++ {
		s.History = append(s.History, &chatdomain.ConversationEntry{
			Prompt:   chatdomain.Message{Role: "user", Content: fmt.Sprintf("Message %d", i+1)},
			Response: chatdomain.Message{Role: "assistant", Content: fmt.Sprintf("Reply %d", i+1)},
		})
	}
	return s
}

func makeSuccessResponse(text string) *ai.Response {
	return &ai.Response{
		ID:     "compact-id",
		Object: "response",
		Output: []ai.ResponseOutputItem{
			{
				Type: "message",
				ID:   "compact-msg",
				Role: "assistant",
				Content: []ai.ResponseOutputContent{
					{Type: "output_text", Text: text},
				},
			},
		},
		Usage: ai.ResponseUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
	}
}

// --- estimateTokens ---

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abcd", 1},
		{"hello world!", 3}, // 12 chars / 4 = 3
	}
	for _, tc := range tests {
		if got := estimateTokens(tc.input); got != tc.want {
			t.Errorf("estimateTokens(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// --- getCompactThreshold ---

func TestGetCompactThreshold_Default(t *testing.T) {
	cs := &CompactService{}
	// No ContextWindowFn → safe default
	if got := cs.getCompactThreshold("basic"); got != 100_000 {
		t.Errorf("default threshold = %d, want 100000", got)
	}
}

func TestGetCompactThreshold_Normal(t *testing.T) {
	cs := &CompactService{
		ContextWindowFn: func(_ string) int { return 128_000 },
	}
	// 128k - 20k buffer = 108k
	if got := cs.getCompactThreshold("basic"); got != 108_000 {
		t.Errorf("threshold = %d, want 108000", got)
	}
}

func TestGetCompactThreshold_Floor(t *testing.T) {
	cs := &CompactService{
		ContextWindowFn: func(_ string) int { return 25_000 },
	}
	// 25k - 20k = 5k, but floor is 10k
	if got := cs.getCompactThreshold("basic"); got != 10_000 {
		t.Errorf("threshold = %d, want 10000 (floor)", got)
	}
}

// --- ShouldCompact ---

func TestShouldCompact_UsesRealTokenCount(t *testing.T) {
	cs := &CompactService{
		ContextWindowFn: func(_ string) int { return 128_000 },
	}
	session := makeSession(2)

	// lastInputTokens = 110_000 > threshold 108_000 → should compact
	if !cs.ShouldCompact(session, "", 110_000) {
		t.Error("expected ShouldCompact=true for high lastInputTokens")
	}
	// lastInputTokens = 50_000 < threshold → should not compact
	if cs.ShouldCompact(session, "", 50_000) {
		t.Error("expected ShouldCompact=false for low lastInputTokens")
	}
}

func TestShouldCompact_FallsBackToEstimation(t *testing.T) {
	cs := &CompactService{
		ContextWindowFn: func(_ string) int { return 200 }, // tiny window for testing
	}
	session := makeSession(2)
	// lastInputTokens=0 → uses estimation; estimated ~6 tokens > threshold 10k? No.
	// With a 200-token window, threshold = floor 10_000, estimated few tokens → false
	if cs.ShouldCompact(session, "", 0) {
		t.Error("expected ShouldCompact=false for tiny estimated tokens vs 10k floor")
	}
}

func TestShouldCompact_CircuitBreaker(t *testing.T) {
	cs := &CompactService{
		ContextWindowFn:     func(_ string) int { return 128_000 },
		consecutiveFailures: maxConsecutiveCompactFailures,
	}
	session := makeSession(2)
	// Even with high token count, circuit breaker prevents compaction
	if cs.ShouldCompact(session, "", 200_000) {
		t.Error("expected ShouldCompact=false when circuit breaker is tripped")
	}
}

// --- Compact ---

func TestCompact_TooFewEntries(t *testing.T) {
	cs := &CompactService{}
	session := makeSession(3) // 3 ≤ compactKeepRecent(4)
	usage, err := cs.Compact(session, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage != nil {
		t.Error("expected nil usage when nothing to compact")
	}
}

func TestCompact_Success(t *testing.T) {
	client := &stubCompactClient{
		response: makeSuccessResponse("This is a summary of the conversation."),
	}
	cs := &CompactService{
		GptClient:       client,
		CostFn:          func(_ string, in, out int) float64 { return float64(in+out) * 0.001 },
		ContextWindowFn: func(_ string) int { return 128_000 },
	}
	session := makeSession(8) // 8 entries: 4 oldest compacted, 4 kept

	usage, err := cs.Compact(session, "")
	if err != nil {
		t.Fatalf("Compact error: %v", err)
	}
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}

	// Should have 1 summary + 4 recent = 5 entries
	if len(session.History) != 5 {
		t.Errorf("history length = %d, want 5", len(session.History))
	}

	// First entry should be the summary
	if !contains(session.History[0].Prompt.Content, "Саммари") {
		t.Errorf("first entry should contain summary marker, got: %q", session.History[0].Prompt.Content)
	}

	// Recent entries preserved
	if session.History[1].Prompt.Content != "Message 5" {
		t.Errorf("first kept entry = %q, want 'Message 5'", session.History[1].Prompt.Content)
	}

	// Cost tracked
	if usage.Cost <= 0 {
		t.Error("expected positive cost")
	}

	if client.calls != 1 {
		t.Errorf("GPT calls = %d, want 1", client.calls)
	}
}

func TestCompact_EmptySummary(t *testing.T) {
	client := &stubCompactClient{
		response: makeSuccessResponse("   "), // whitespace only
	}
	cs := &CompactService{GptClient: client}
	session := makeSession(8)
	originalLen := len(session.History)

	usage, err := cs.Compact(session, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage != nil {
		t.Error("expected nil usage for empty summary")
	}
	// History should be unchanged
	if len(session.History) != originalLen {
		t.Errorf("history modified: %d → %d", originalLen, len(session.History))
	}
	// consecutiveFailures should increment
	if cs.consecutiveFailures != 1 {
		t.Errorf("consecutiveFailures = %d, want 1", cs.consecutiveFailures)
	}
}

func TestCompact_GPTError_CircuitBreaker(t *testing.T) {
	client := &stubCompactClient{err: errors.New("API error")}
	cs := &CompactService{GptClient: client}
	session := makeSession(8)

	// Three consecutive failures should trip the circuit breaker
	for i := 0; i < maxConsecutiveCompactFailures; i++ {
		_, err := cs.Compact(session, "")
		if err == nil {
			t.Fatalf("expected error on call %d", i+1)
		}
	}

	if cs.consecutiveFailures != maxConsecutiveCompactFailures {
		t.Errorf("consecutiveFailures = %d, want %d", cs.consecutiveFailures, maxConsecutiveCompactFailures)
	}
}

func TestCompact_SuccessResetsCircuitBreaker(t *testing.T) {
	client := &stubCompactClient{
		response: makeSuccessResponse("Summary after failures."),
	}
	cs := &CompactService{
		GptClient:           client,
		consecutiveFailures: 2, // simulate previous failures
	}
	session := makeSession(8)

	_, err := cs.Compact(session, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cs.consecutiveFailures != 0 {
		t.Errorf("consecutiveFailures = %d, want 0 after success", cs.consecutiveFailures)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
