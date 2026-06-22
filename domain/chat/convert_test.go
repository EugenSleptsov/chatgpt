package chat

import (
	"GPTBot/domain/ai"
	"testing"
)

func TestToGPTMessages(t *testing.T) {
	entries := []*ConversationEntry{
		{
			Prompt:   Message{Role: "user", Content: "hello"},
			Response: Message{Role: "assistant", Content: "hi"},
		},
		{
			Prompt:   Message{Role: "user", Content: "bye"},
			Response: Message{}, // empty response
		},
	}
	msgs := ToGPTMessages(entries)
	// First entry: prompt + response = 2
	// Second entry: prompt only = 1 (empty response skipped)
	if len(msgs) != 3 {
		t.Fatalf("len = %d, want 3", len(msgs))
	}
	if msgs[0] != (ai.Message{Role: "user", Content: "hello"}) {
		t.Errorf("msgs[0] = %+v", msgs[0])
	}
	if msgs[1] != (ai.Message{Role: "assistant", Content: "hi"}) {
		t.Errorf("msgs[1] = %+v", msgs[1])
	}
	if msgs[2] != (ai.Message{Role: "user", Content: "bye"}) {
		t.Errorf("msgs[2] = %+v", msgs[2])
	}
}

func TestToGPTMessages_Empty(t *testing.T) {
	msgs := ToGPTMessages(nil)
	if len(msgs) != 0 {
		t.Fatalf("expected empty, got %d", len(msgs))
	}
}
