package service

import (
	chatdomain "GPTBot/domain/chat"
	"strings"
	"testing"
	"time"
)

func TestAddAdvisorNote_CreatesTopicAndAppends(t *testing.T) {
	chat := &chatdomain.Chat{}

	topic, err := AddAdvisorNote(chat, "Налоги", "подать декларацию до мая", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic.ID != 1 || topic.Name != "Налоги" || len(topic.Entries) != 1 {
		t.Fatalf("unexpected topic state: %+v", topic)
	}

	// Same topic name (different case) must append, not duplicate.
	_, err = AddAdvisorNote(chat, "налоги", "проверить вычеты", nil)
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
	other, _ := AddAdvisorNote(chat, "ИКЕА", "купить полки", nil)
	if other.ID != 2 || len(chat.AdvisorTopics) != 2 {
		t.Fatalf("expected second topic with ID 2, got %+v", other)
	}
}

func TestAddAdvisorNote_RejectsEmpty(t *testing.T) {
	chat := &chatdomain.Chat{}
	if _, err := AddAdvisorNote(chat, "", "note", nil); err == nil {
		t.Fatal("expected error for empty topic")
	}
	if _, err := AddAdvisorNote(chat, "Тема", "  ", nil); err == nil {
		t.Fatal("expected error for empty note")
	}
}

func TestRemoveAdvisorEntry_DropsEmptyTopic(t *testing.T) {
	chat := &chatdomain.Chat{}
	topic, _ := AddAdvisorNote(chat, "Налоги", "одна запись", nil)

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
	src, _ := AddAdvisorNote(chat, "Покупки", "молоко", nil)
	AddAdvisorNote(chat, "Покупки", "хлеб", nil)
	dst, _ := AddAdvisorNote(chat, "ИКЕА", "полки", nil)

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

func TestParseRemindAt(t *testing.T) {
	if at, err := ParseRemindAt(""); err != nil || at != nil {
		t.Fatalf("empty input: at=%v err=%v, want nil/nil", at, err)
	}

	at, err := ParseRemindAt("2026-07-10 15:30")
	if err != nil || at == nil {
		t.Fatalf("datetime parse failed: %v", err)
	}
	if at.Hour() != 15 || at.Minute() != 30 || at.Day() != 10 {
		t.Fatalf("parsed wrong time: %v", at)
	}

	at, err = ParseRemindAt("2026-07-10")
	if err != nil || at == nil {
		t.Fatalf("date parse failed: %v", err)
	}
	if at.Hour() != defaultRemindHour || at.Minute() != 0 {
		t.Fatalf("bare date must default to %d:00, got %v", defaultRemindHour, at)
	}

	if _, err := ParseRemindAt("завтра"); err == nil {
		t.Fatal("expected error for unparseable input")
	}
}

func TestAddAdvisorNote_WithReminder(t *testing.T) {
	chat := &chatdomain.Chat{}
	at := time.Date(2026, 7, 10, 15, 0, 0, 0, time.Local)

	topic, err := AddAdvisorNote(chat, "Налоги", "написать адвокату", &at)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	e := topic.Entries[0]
	if e.RemindAt == nil || !e.RemindAt.Equal(at) {
		t.Fatalf("RemindAt = %v, want %v", e.RemindAt, at)
	}

	// read_notes output must expose the ID and the reminder.
	out := AdvisorNotesForTool(topic)
	if !strings.Contains(out, "#1") || !strings.Contains(out, "2026-07-10 15:00") {
		t.Fatalf("notes output must contain entry ID and reminder time, got: %s", out)
	}
}

func TestSetAdvisorReminder_SetMoveClear(t *testing.T) {
	chat := &chatdomain.Chat{}
	topic, _ := AddAdvisorNote(chat, "Налоги", "написать адвокату", nil)
	eid := topic.Entries[0].ID

	at := time.Now().Add(time.Hour)
	if !chat.SetAdvisorReminder(topic.ID, eid, &at) {
		t.Fatal("set reminder must succeed")
	}
	if topic.Entries[0].RemindAt == nil {
		t.Fatal("RemindAt must be set")
	}
	if !chat.SetAdvisorReminder(topic.ID, eid, nil) {
		t.Fatal("clear reminder must succeed")
	}
	if topic.Entries[0].RemindAt != nil {
		t.Fatal("RemindAt must be cleared")
	}
	if chat.SetAdvisorReminder(topic.ID, 999, &at) || chat.SetAdvisorReminder(999, eid, &at) {
		t.Fatal("missing entry/topic must return false")
	}
}

func TestDueAdvisorReminders(t *testing.T) {
	chat := &chatdomain.Chat{}
	past := time.Now().Add(-time.Minute)
	future := time.Now().Add(time.Hour)
	AddAdvisorNote(chat, "Налоги", "просрочено", &past)
	AddAdvisorNote(chat, "Налоги", "еще рано", &future)
	AddAdvisorNote(chat, "Налоги", "без напоминания", nil)

	due := chat.DueAdvisorReminders(time.Now())
	if len(due) != 1 || due[0].Entry.Text != "просрочено" {
		t.Fatalf("expected exactly the overdue entry, got %+v", due)
	}
}

func TestAdvisorPrompt(t *testing.T) {
	chat := &chatdomain.Chat{}
	if AdvisorPrompt(chat) != "" {
		t.Fatal("expected empty prompt for no topics")
	}
	AddAdvisorNote(chat, "Налоги", "декларация", nil)
	AddAdvisorNote(chat, "Налоги", "вычеты", nil)
	got := AdvisorPrompt(chat)
	if !strings.Contains(got, "Налоги (2)") {
		t.Fatalf("prompt must list topic with count, got: %s", got)
	}
}
