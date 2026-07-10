package service

import (
	chatdomain "GPTBot/domain/chat"
	"GPTBot/infrastructure/storage"
	"strings"
	"testing"
)

func supermemoryChat() *chatdomain.Chat {
	return &chatdomain.Chat{
		ChatID:   1,
		Settings: chatdomain.ChatSettings{Supermemory: true},
	}
}

func TestArchiveHistory_ArchivesAndStampsRanges(t *testing.T) {
	archive := storage.NewMemoryArchive()
	chat := supermemoryChat()
	session := &chatdomain.Session{ID: 1}

	AppendHistory(session, chatdomain.Message{Role: "user", Content: "привет"})
	AttachResponse(session, chatdomain.Message{Role: "assistant", Content: "здравствуйте"})
	ArchiveHistory(archive, chat, session)

	e := session.History[0]
	if e.ArchiveFrom != 0 || e.ArchiveTo != 2 {
		t.Fatalf("entry range = [%d, %d), want [0, 2)", e.ArchiveFrom, e.ArchiveTo)
	}
	msgs, err := archive.ReadRange(1, 0, 2)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("ReadRange: %v, %d msgs", err, len(msgs))
	}
	if msgs[0].Content != "привет" || msgs[1].Role != "assistant" {
		t.Fatalf("unexpected archive content: %+v", msgs)
	}

	// Re-running must not duplicate lines.
	ArchiveHistory(archive, chat, session)
	if _, err := archive.ReadRange(1, 2, 3); err == nil {
		t.Fatal("expected no third line after repeated ArchiveHistory")
	}
}

func TestArchiveHistory_ResponseAttachedLater(t *testing.T) {
	archive := storage.NewMemoryArchive()
	chat := supermemoryChat()
	session := &chatdomain.Session{ID: 1}

	AppendHistory(session, chatdomain.Message{Role: "user", Content: "вопрос"})
	ArchiveHistory(archive, chat, session) // prompt only → range [0, 1)

	AttachResponse(session, chatdomain.Message{Role: "assistant", Content: "ответ"})
	ArchiveHistory(archive, chat, session) // response line appended → [0, 2)

	e := session.History[0]
	if e.ArchiveFrom != 0 || e.ArchiveTo != 2 {
		t.Fatalf("entry range = [%d, %d), want [0, 2)", e.ArchiveFrom, e.ArchiveTo)
	}
	msgs, _ := archive.ReadRange(1, 0, 2)
	if len(msgs) != 2 || msgs[1].Content != "ответ" {
		t.Fatalf("unexpected archive content: %+v", msgs)
	}
}

func TestArchiveHistory_DisabledIsNoop(t *testing.T) {
	archive := storage.NewMemoryArchive()
	chat := &chatdomain.Chat{ChatID: 1} // supermemory off
	session := &chatdomain.Session{ID: 1}
	AppendHistory(session, chatdomain.Message{Role: "user", Content: "привет"})

	ArchiveHistory(archive, chat, session)
	if session.History[0].ArchiveTo != 0 {
		t.Fatal("nothing must be archived when supermemory is off")
	}
}

func TestSupermemoryPrompt(t *testing.T) {
	chat := supermemoryChat()
	if SupermemoryPrompt(chat) != "" {
		t.Fatal("empty prompt expected for no nodes")
	}
	chat.AddMemoryNode("обсуждение налогов", "детали...", 0, 4, 1)
	got := SupermemoryPrompt(chat)
	if !strings.Contains(got, "#1 — обсуждение налогов") {
		t.Fatalf("index must list the node hook, got: %q", got)
	}

	chat.Settings.Supermemory = false
	if SupermemoryPrompt(chat) != "" {
		t.Fatal("prompt must be empty when supermemory is off (nodes kept)")
	}
	if len(chat.MemoryNodes) != 1 {
		t.Fatal("disabling must not delete nodes")
	}
}

func TestSupermemoryPrompt_RootsOnly(t *testing.T) {
	chat := supermemoryChat()
	a := chat.AddMemoryNode("child a", "…", 0, 0, 1)
	b := chat.AddMemoryNode("child b", "…", 0, 0, 1)
	parent := chat.AddMemoryNode("parent hook", "merged summary", 0, 0, 1)
	parent.Children = []int{a.ID, b.ID}

	got := SupermemoryPrompt(chat)
	if strings.Contains(got, "child a") || !strings.Contains(got, "parent hook") {
		t.Fatalf("index must show only roots, got: %q", got)
	}
}

func TestSearchMemoryNodes(t *testing.T) {
	chat := supermemoryChat()
	chat.AddMemoryNode("Налоги 2026", "обсуждали декларацию и вычеты", 0, 2, 1)
	chat.AddMemoryNode("Ремонт", "плитка для ванной", 2, 4, 1)

	if got := SearchMemoryNodes(chat, "декларац"); len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("summary search failed: %+v", got)
	}
	if got := SearchMemoryNodes(chat, "РЕМОНТ"); len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("case-insensitive hook search failed: %+v", got)
	}
	if got := SearchMemoryNodes(chat, "квартира"); got != nil {
		t.Fatalf("expected no matches, got %+v", got)
	}
	if got := SearchMemoryNodes(chat, "  "); got != nil {
		t.Fatalf("empty query must return nil, got %+v", got)
	}
}

func TestSplitHookSummary(t *testing.T) {
	hook, body := splitHookSummary("Заголовок\n\nТело саммари.")
	if hook != "Заголовок" || body != "Тело саммари." {
		t.Fatalf("got hook=%q body=%q", hook, body)
	}
	// Model ignored the format: single paragraph.
	hook, body = splitHookSummary("Одна строка без переноса")
	if hook != "Одна строка без переноса" || body != "Одна строка без переноса" {
		t.Fatalf("fallback failed: hook=%q body=%q", hook, body)
	}
}

func TestArchiveRangeOfEntries(t *testing.T) {
	entries := []*chatdomain.ConversationEntry{
		{ArchiveFrom: 4, ArchiveTo: 6},
		{}, // not archived
		{ArchiveFrom: 6, ArchiveTo: 8},
	}
	from, to := archiveRangeOfEntries(entries)
	if from != 4 || to != 8 {
		t.Fatalf("range = [%d, %d), want [4, 8)", from, to)
	}
	if f, tt := archiveRangeOfEntries([]*chatdomain.ConversationEntry{{}}); f != 0 || tt != 0 {
		t.Fatalf("unarchived entries must yield zero range, got [%d, %d)", f, tt)
	}
}
