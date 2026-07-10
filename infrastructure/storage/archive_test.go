package storage

import (
	"GPTBot/domain/chat"
	"testing"
	"time"
)

func sampleMsgs(contents ...string) []chat.ArchivedMessage {
	msgs := make([]chat.ArchivedMessage, 0, len(contents))
	for _, c := range contents {
		msgs = append(msgs, chat.ArchivedMessage{Time: time.Now(), SessionID: 1, Role: "user", Content: c})
	}
	return msgs
}

func TestFileArchive_AppendAndReadRange(t *testing.T) {
	a, err := NewFileArchive(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	from, err := a.Append(7, sampleMsgs("один", "два"))
	if err != nil || from != 0 {
		t.Fatalf("first append: from=%d err=%v", from, err)
	}
	from, err = a.Append(7, sampleMsgs("три"))
	if err != nil || from != 2 {
		t.Fatalf("second append: from=%d err=%v", from, err)
	}

	msgs, err := a.ReadRange(7, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Content != "два" || msgs[1].Content != "три" {
		t.Fatalf("unexpected range content: %+v", msgs)
	}
}

func TestFileArchive_CountSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	a, _ := NewFileArchive(dir)
	if _, err := a.Append(7, sampleMsgs("один", "два")); err != nil {
		t.Fatal(err)
	}

	// New instance (fresh cache) must continue numbering from the file.
	b, _ := NewFileArchive(dir)
	from, err := b.Append(7, sampleMsgs("три"))
	if err != nil || from != 2 {
		t.Fatalf("append after reopen: from=%d err=%v", from, err)
	}
}

func TestFileArchive_PerChatIsolation(t *testing.T) {
	a, _ := NewFileArchive(t.TempDir())
	_, _ = a.Append(1, sampleMsgs("чат один"))
	from, _ := a.Append(2, sampleMsgs("чат два"))
	if from != 0 {
		t.Fatalf("chats must have independent line numbering, got from=%d", from)
	}
	msgs, err := a.ReadRange(2, 0, 1)
	if err != nil || len(msgs) != 1 || msgs[0].Content != "чат два" {
		t.Fatalf("cross-chat leak: %+v err=%v", msgs, err)
	}
}

func TestFileArchive_DeleteRangeStub(t *testing.T) {
	a, _ := NewFileArchive(t.TempDir())
	if err := a.DeleteRange(1, 0, 1); err == nil {
		t.Fatal("DeleteRange is a reserved stub and must error until implemented")
	}
}

func TestMemoryArchive_Bounds(t *testing.T) {
	a := NewMemoryArchive()
	_, _ = a.Append(1, sampleMsgs("один"))
	if _, err := a.ReadRange(1, 0, 2); err == nil {
		t.Fatal("out-of-bounds read must error")
	}
	msgs, err := a.ReadRange(1, 0, 1)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("valid read failed: %v", err)
	}
}
