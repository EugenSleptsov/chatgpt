package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"GPTBot/infrastructure/storage"
	"errors"
	"fmt"
	"strings"
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
func (s *stubCompactClient) GenerateImage(_ string, _ string) ([]byte, error) { return nil, nil }
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
	usage, err := cs.Compact(&chatdomain.Chat{}, session, "")
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

	usage, err := cs.Compact(&chatdomain.Chat{}, session, "")
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

func TestCompact_SummaryEntryInheritsArchiveRange(t *testing.T) {
	client := &stubCompactClient{
		response: makeSuccessResponse("Саммари разговора."),
	}
	cs := &CompactService{
		GptClient:       client,
		ContextWindowFn: func(_ string) int { return 128_000 },
	}
	chat := &chatdomain.Chat{Settings: chatdomain.ChatSettings{Supermemory: true}}
	session := makeSession(8)
	// Simulate ingest-time archiving: entries 0..7 → lines 0..15.
	for i, e := range session.History {
		e.ArchiveFrom, e.ArchiveTo = i*2, i*2+2
	}

	if _, err := cs.Compact(chat, session, ""); err != nil {
		t.Fatalf("Compact error: %v", err)
	}

	// Compaction no longer creates nodes — snapshots do.
	if len(chat.MemoryNodes) != 0 {
		t.Fatalf("compaction must not create nodes, got %d", len(chat.MemoryNodes))
	}
	// Summary entry inherits the evicted range so it is never re-archived.
	if session.History[0].ArchiveFrom != 0 || session.History[0].ArchiveTo != 8 {
		t.Errorf("summary entry range = [%d, %d), want [0, 8)", session.History[0].ArchiveFrom, session.History[0].ArchiveTo)
	}
}

// --- Snapshot ---

func snapshotFixture(t *testing.T, lines int, contentSize int) (*CompactService, *chatdomain.Chat, *stubCompactClient) {
	t.Helper()
	client := &stubCompactClient{
		response: makeSuccessResponse("Хук снапшота\n\nСаммари куска переписки."),
	}
	archive := storage.NewMemoryArchive()
	chat := &chatdomain.Chat{ChatID: 1, Settings: chatdomain.ChatSettings{Supermemory: true}}
	filler := strings.Repeat("х", contentSize)
	for i := 0; i < lines; i++ {
		if _, err := archive.Append(1, []chatdomain.ArchivedMessage{{Role: "user", Content: filler}}); err != nil {
			t.Fatal(err)
		}
	}
	return &CompactService{GptClient: client, Archive: archive}, chat, client
}

func TestSnapshot_BelowThresholdWaits(t *testing.T) {
	cs, chat, client := snapshotFixture(t, 4, 100) // ~100 tokens total — far below threshold
	usage, err := cs.Snapshot(chat, "basic", false)
	if err != nil || usage != nil {
		t.Fatalf("expected noop below threshold, usage=%v err=%v", usage, err)
	}
	if client.calls != 0 || len(chat.MemoryNodes) != 0 {
		t.Fatal("no GPT call and no node expected below threshold")
	}
}

func TestSnapshot_OverThresholdCreatesNode(t *testing.T) {
	// 4 lines × 4000 chars ≈ 4000 tokens > 3000 threshold.
	cs, chat, _ := snapshotFixture(t, 4, 4000)
	usage, err := cs.Snapshot(chat, "basic", false)
	if err != nil || usage == nil {
		t.Fatalf("expected a snapshot, usage=%v err=%v", usage, err)
	}
	if len(chat.MemoryNodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(chat.MemoryNodes))
	}
	n := chat.MemoryNodes[0]
	if n.Hook != "Хук снапшота" || n.From != 0 || n.To != 4 {
		t.Fatalf("node = %+v, want hook 'Хук снапшота' range [0, 4)", n)
	}
	if chat.ArchiveSnapshotTo != 4 {
		t.Fatalf("pointer = %d, want 4", chat.ArchiveSnapshotTo)
	}

	// Nothing uncovered left — force must be a noop too.
	if usage, _ := cs.Snapshot(chat, "basic", true); usage != nil {
		t.Fatal("covered archive must not be re-snapshotted")
	}
}

func TestSnapshot_ForceIgnoresThreshold(t *testing.T) {
	cs, chat, _ := snapshotFixture(t, 4, 100)
	usage, err := cs.Snapshot(chat, "basic", true)
	if err != nil || usage == nil {
		t.Fatalf("force must snapshot below threshold, usage=%v err=%v", usage, err)
	}
	if len(chat.MemoryNodes) != 1 || chat.ArchiveSnapshotTo != 4 {
		t.Fatalf("node/pointer wrong: nodes=%d pointer=%d", len(chat.MemoryNodes), chat.ArchiveSnapshotTo)
	}
}

func TestSnapshot_ForceSkipsTrivialTail(t *testing.T) {
	cs, chat, client := snapshotFixture(t, 1, 100) // single line — not worth a node
	if usage, _ := cs.Snapshot(chat, "basic", true); usage != nil {
		t.Fatal("a lone line must not become a node")
	}
	if client.calls != 0 {
		t.Fatal("no GPT call expected for a trivial tail")
	}
}

func TestSnapshot_DisabledIsNoop(t *testing.T) {
	cs, chat, client := snapshotFixture(t, 4, 4000)
	chat.Settings.Supermemory = false
	if usage, _ := cs.Snapshot(chat, "basic", true); usage != nil {
		t.Fatal("expected noop when disabled")
	}
	if client.calls != 0 {
		t.Fatal("no GPT call expected when disabled")
	}
}

// --- MetaCompact ---

func metaChatWithRoots(n int) *chatdomain.Chat {
	chat := &chatdomain.Chat{Settings: chatdomain.ChatSettings{Supermemory: true}}
	for i := 0; i < n; i++ {
		chat.AddMemoryNode(fmt.Sprintf("hook %d", i+1), fmt.Sprintf("summary %d", i+1), i*2, i*2+2, 1)
	}
	return chat
}

func TestMetaCompact_FoldsOldestIntoParent(t *testing.T) {
	client := &stubCompactClient{
		response: makeSuccessResponse("Общий хук\n\nОбъединённое саммари."),
	}
	cs := &CompactService{GptClient: client}
	chat := metaChatWithRoots(supermemoryIndexLimit + 1) // 41 roots — over budget

	usage, err := cs.MetaCompact(chat, "basic")
	if err != nil {
		t.Fatalf("MetaCompact error: %v", err)
	}
	if usage == nil {
		t.Fatal("expected usage for a performed merge")
	}

	parent := chat.MemoryNodes[len(chat.MemoryNodes)-1]
	if parent.Hook != "Общий хук" || len(parent.Children) != metaCompactBatch {
		t.Fatalf("parent = %+v, want hook 'Общий хук' with %d children", parent, metaCompactBatch)
	}
	// Children must be the oldest roots (#1..#10) and disappear from the index.
	if parent.Children[0] != 1 || parent.Children[metaCompactBatch-1] != metaCompactBatch {
		t.Fatalf("children = %v, want oldest IDs 1..%d", parent.Children, metaCompactBatch)
	}
	roots := chat.RootMemoryNodes()
	want := supermemoryIndexLimit + 1 - metaCompactBatch + 1 // folded 10, added 1 parent
	if len(roots) != want {
		t.Fatalf("roots after merge = %d, want %d", len(roots), want)
	}
	// Bodies untouched.
	if chat.FindMemoryNode(1).Summary != "summary 1" {
		t.Fatal("child summaries must never be rewritten")
	}
}

func TestMetaCompact_NoopUnderBudget(t *testing.T) {
	client := &stubCompactClient{response: makeSuccessResponse("x\n\ny")}
	cs := &CompactService{GptClient: client}
	chat := metaChatWithRoots(supermemoryIndexLimit) // exactly at budget

	usage, err := cs.MetaCompact(chat, "basic")
	if err != nil || usage != nil {
		t.Fatalf("expected noop, got usage=%v err=%v", usage, err)
	}
	if client.calls != 0 {
		t.Fatal("no GPT call expected under budget")
	}
}

func TestMetaCompact_SkipsPinned(t *testing.T) {
	client := &stubCompactClient{
		response: makeSuccessResponse("Хук\n\nСаммари."),
	}
	cs := &CompactService{GptClient: client}
	chat := metaChatWithRoots(supermemoryIndexLimit + 1)
	chat.FindMemoryNode(1).Pinned = true

	if _, err := cs.MetaCompact(chat, "basic"); err != nil {
		t.Fatal(err)
	}
	parent := chat.MemoryNodes[len(chat.MemoryNodes)-1]
	for _, id := range parent.Children {
		if id == 1 {
			t.Fatal("pinned node must not be folded")
		}
	}
}

func TestMetaCompact_DisabledIsNoop(t *testing.T) {
	client := &stubCompactClient{response: makeSuccessResponse("x\n\ny")}
	cs := &CompactService{GptClient: client}
	chat := metaChatWithRoots(supermemoryIndexLimit + 1)
	chat.Settings.Supermemory = false

	if usage, err := cs.MetaCompact(chat, "basic"); usage != nil || err != nil {
		t.Fatalf("expected noop when disabled, got usage=%v err=%v", usage, err)
	}
}

func TestCompact_EmptySummary(t *testing.T) {
	client := &stubCompactClient{
		response: makeSuccessResponse("   "), // whitespace only
	}
	cs := &CompactService{GptClient: client}
	session := makeSession(8)
	originalLen := len(session.History)

	usage, err := cs.Compact(&chatdomain.Chat{}, session, "")
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
		_, err := cs.Compact(&chatdomain.Chat{}, session, "")
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

	_, err := cs.Compact(&chatdomain.Chat{}, session, "")
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
