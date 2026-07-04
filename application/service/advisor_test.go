package service

import (
	chatdomain "GPTBot/domain/chat"
	"strings"
	"testing"
)

func TestAddAdvisorNote_CreatesTopicAndAppends(t *testing.T) {
	chat := &chatdomain.Chat{}

	topic, err := AddAdvisorNote(chat, "Налоги", "подать декларацию до мая")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic.ID != 1 || topic.Name != "Налоги" || len(topic.Entries) != 1 {
		t.Fatalf("unexpected topic state: %+v", topic)
	}

	// Same topic name (different case) must append, not duplicate.
	_, err = AddAdvisorNote(chat, "налоги", "проверить вычеты")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chat.AdvisorTopics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(chat.AdvisorTopics))
	}
	if len(topic.Entries) != 2 || topic.Entries[1].ID != 2 {
		t.Fatalf("expected 2 entries with sequential IDs, got %+v", topic.Entries)
	}

	// New name creates a second topic with the next ID.
	other, _ := AddAdvisorNote(chat, "ИКЕА", "купить полки")
	if other.ID != 2 || len(chat.AdvisorTopics) != 2 {
		t.Fatalf("expected second topic with ID 2, got %+v", other)
	}
}

func TestAddAdvisorNote_RejectsEmpty(t *testing.T) {
	chat := &chatdomain.Chat{}
	if _, err := AddAdvisorNote(chat, "", "note"); err == nil {
		t.Fatal("expected error for empty topic")
	}
	if _, err := AddAdvisorNote(chat, "Тема", "  "); err == nil {
		t.Fatal("expected error for empty note")
	}
}

func TestRemoveAdvisorEntry_DropsEmptyTopic(t *testing.T) {
	chat := &chatdomain.Chat{}
	topic, _ := AddAdvisorNote(chat, "Налоги", "одна запись")

	if !chat.RemoveAdvisorEntry(topic.ID, topic.Entries[0].ID) {
		t.Fatal("expected removal to succeed")
	}
	if len(chat.AdvisorTopics) != 0 {
		t.Fatalf("empty topic must be removed, got %d topics", len(chat.AdvisorTopics))
	}
	if chat.RemoveAdvisorEntry(999, 1) {
		t.Fatal("expected false for missing topic")
	}
}

func TestMergeAdvisorTopics(t *testing.T) {
	chat := &chatdomain.Chat{}
	src, _ := AddAdvisorNote(chat, "Покупки", "молоко")
	AddAdvisorNote(chat, "Покупки", "хлеб")
	dst, _ := AddAdvisorNote(chat, "ИКЕА", "полки")

	if !chat.MergeAdvisorTopics(src.ID, dst.ID) {
		t.Fatal("expected merge to succeed")
	}
	if len(chat.AdvisorTopics) != 1 {
		t.Fatalf("expected 1 topic after merge, got %d", len(chat.AdvisorTopics))
	}
	if len(dst.Entries) != 3 {
		t.Fatalf("expected 3 entries in destination, got %d", len(dst.Entries))
	}
	// Entry IDs must stay unique within the merged topic.
	seen := map[int]bool{}
	for _, e := range dst.Entries {
		if seen[e.ID] {
			t.Fatalf("duplicate entry ID %d after merge", e.ID)
		}
		seen[e.ID] = true
	}
	if chat.MergeAdvisorTopics(dst.ID, dst.ID) {
		t.Fatal("merging a topic into itself must fail")
	}
}

func TestAdvisorPrompt(t *testing.T) {
	chat := &chatdomain.Chat{}
	if AdvisorPrompt(chat) != "" {
		t.Fatal("expected empty prompt for no topics")
	}
	AddAdvisorNote(chat, "Налоги", "декларация")
	AddAdvisorNote(chat, "Налоги", "вычеты")
	got := AdvisorPrompt(chat)
	if !strings.Contains(got, "Налоги (2)") {
		t.Fatalf("prompt must list topic with count, got: %s", got)
	}
}
