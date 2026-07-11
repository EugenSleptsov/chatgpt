package service

import (
	chatdomain "GPTBot/domain/chat"
	"strings"
	"testing"
)

// makeDreamChat builds a supermemory-enabled chat with n root nodes.
func makeDreamChat(n int) *chatdomain.Chat {
	ch := &chatdomain.Chat{ChatID: 1}
	ch.Settings.Supermemory = true
	for i := 0; i < n; i++ {
		ch.AddMemoryNode("hook", "summary", 0, 0, 0)
	}
	return ch
}

func TestDream_AppliesPlan(t *testing.T) {
	ch := makeDreamChat(5)
	client := &stubCompactClient{response: makeSuccessResponse(
		`{"groups":[{"hook":"Тема","summary":"Общее саммари","node_ids":[1,2,99]}]}`)}
	cs := &CompactService{GptClient: client}

	created, usage, err := cs.Dream(ch, "basic")
	if err != nil {
		t.Fatalf("Dream error: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %d nodes, want 1", len(created))
	}
	parent := created[0]
	if parent.Hook != "Тема" || parent.Summary != "Общее саммари" {
		t.Errorf("parent = %q / %q", parent.Hook, parent.Summary)
	}
	// id 99 does not exist and must be dropped from the group.
	if len(parent.Children) != 2 || parent.Children[0] != 1 || parent.Children[1] != 2 {
		t.Errorf("children = %v, want [1 2]", parent.Children)
	}
	// Grouped nodes are no longer roots; the other three plus the parent are.
	roots := ch.RootMemoryNodes()
	if len(roots) != 4 {
		t.Errorf("roots = %d, want 4", len(roots))
	}
	if usage == nil {
		t.Error("usage = nil, want tracked usage")
	}
}

func TestDream_EmptyPlanCreatesNothing(t *testing.T) {
	ch := makeDreamChat(5)
	client := &stubCompactClient{response: makeSuccessResponse(`{"groups":[]}`)}
	cs := &CompactService{GptClient: client}

	created, _, err := cs.Dream(ch, "basic")
	if err != nil {
		t.Fatalf("Dream error: %v", err)
	}
	if len(created) != 0 || len(ch.MemoryNodes) != 5 {
		t.Errorf("created = %d, nodes = %d — want no changes", len(created), len(ch.MemoryNodes))
	}
}

func TestDream_TooFewRootsSkipsGPT(t *testing.T) {
	ch := makeDreamChat(3)
	client := &stubCompactClient{response: makeSuccessResponse(`{"groups":[]}`)}
	cs := &CompactService{GptClient: client}

	created, usage, err := cs.Dream(ch, "basic")
	if err != nil || created != nil || usage != nil {
		t.Errorf("Dream = (%v, %v, %v), want full no-op", created, usage, err)
	}
	if client.calls != 0 {
		t.Errorf("CallGPT invoked %d times, want 0", client.calls)
	}
}

func TestDream_PinnedNodesNotRegrouped(t *testing.T) {
	ch := makeDreamChat(6)
	ch.FindMemoryNode(1).Pinned = true
	client := &stubCompactClient{response: makeSuccessResponse(
		`{"groups":[{"hook":"Тема","summary":"s","node_ids":[1,2,3]}]}`)}
	cs := &CompactService{GptClient: client}

	created, _, err := cs.Dream(ch, "basic")
	if err != nil {
		t.Fatalf("Dream error: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %d nodes, want 1", len(created))
	}
	// The pinned node must be filtered out of the group.
	if len(created[0].Children) != 2 || created[0].Children[0] != 2 || created[0].Children[1] != 3 {
		t.Errorf("children = %v, want [2 3]", created[0].Children)
	}
}

func TestDream_UndersizedGroupSkipped(t *testing.T) {
	ch := makeDreamChat(5)
	// One valid id + one bogus id → effective group of 1 → skipped.
	client := &stubCompactClient{response: makeSuccessResponse(
		`{"groups":[{"hook":"Тема","summary":"s","node_ids":[1,42]}]}`)}
	cs := &CompactService{GptClient: client}

	created, _, err := cs.Dream(ch, "basic")
	if err != nil {
		t.Fatalf("Dream error: %v", err)
	}
	if len(created) != 0 || len(ch.MemoryNodes) != 5 {
		t.Errorf("created = %d, nodes = %d — want group skipped", len(created), len(ch.MemoryNodes))
	}
}

func TestDream_BadJSONReturnsError(t *testing.T) {
	ch := makeDreamChat(5)
	client := &stubCompactClient{response: makeSuccessResponse("I refuse to answer in JSON.")}
	cs := &CompactService{GptClient: client}

	if _, _, err := cs.Dream(ch, "basic"); err == nil {
		t.Error("Dream error = nil, want parse error")
	}
	if len(ch.MemoryNodes) != 5 {
		t.Errorf("nodes = %d, want unchanged 5", len(ch.MemoryNodes))
	}
}

func TestDream_DisabledIsNoop(t *testing.T) {
	ch := makeDreamChat(6)
	ch.Settings.Supermemory = false
	client := &stubCompactClient{response: makeSuccessResponse(`{"groups":[]}`)}
	cs := &CompactService{GptClient: client}

	created, usage, err := cs.Dream(ch, "basic")
	if err != nil || created != nil || usage != nil || client.calls != 0 {
		t.Errorf("Dream on disabled chat must be a full no-op (calls=%d)", client.calls)
	}
}

func TestParseDreamPlan_ToleratesFences(t *testing.T) {
	groups, err := parseDreamPlan("```json\n{\"groups\":[{\"hook\":\"h\",\"summary\":\"s\",\"node_ids\":[1,2]}]}\n```")
	if err != nil {
		t.Fatalf("parseDreamPlan error: %v", err)
	}
	if len(groups) != 1 || len(groups[0].NodeIDs) != 2 {
		t.Errorf("groups = %+v", groups)
	}
}

func TestGPTServiceDream_StampsLastDreamNodeID(t *testing.T) {
	ch := makeDreamChat(5)
	client := &stubCompactClient{response: makeSuccessResponse(
		`{"groups":[{"hook":"Тема","summary":"s","node_ids":[1,2]}]}`)}
	svc := &GPTService{GptClient: client, Compact: &CompactService{GptClient: client}}

	report, created := svc.Dream(ch)
	if created != 1 || !strings.Contains(report, "Тема") {
		t.Fatalf("Dream = (%q, %d)", report, created)
	}
	if ch.LastDreamNodeID != ch.NextMemoryNodeID {
		t.Errorf("LastDreamNodeID = %d, want %d (stamped after run)", ch.LastDreamNodeID, ch.NextMemoryNodeID)
	}
}

func TestGPTServiceDream_TooFewDoesNotStamp(t *testing.T) {
	ch := makeDreamChat(2)
	svc := &GPTService{Compact: &CompactService{GptClient: &stubCompactClient{}}}

	report, created := svc.Dream(ch)
	if created != 0 || !strings.Contains(report, "мало") {
		t.Fatalf("Dream = (%q, %d)", report, created)
	}
	if ch.LastDreamNodeID != 0 {
		t.Errorf("LastDreamNodeID = %d, want 0 (no run happened)", ch.LastDreamNodeID)
	}
}
