package storage

import (
	"GPTBot/domain/chat"
	"os"
	"path/filepath"
	"testing"
)

// ===================== ActiveSession =====================

func newChat() *chat.Chat {
	return &chat.Chat{
		ChatID: 1,
		Sessions: []*chat.Session{
			{ID: 1, Topic: "first", History: make([]*chat.ConversationEntry, 0), Model: "basic"},
			{ID: 2, Topic: "second", History: make([]*chat.ConversationEntry, 0), Model: "fast"},
		},
		ActiveSessionID: 1,
		NextSessionID:   3,
	}
}

func TestActiveSession_ReturnsActive(t *testing.T) {
	c := newChat()
	s := c.ActiveSession()
	if s.ID != 1 || s.Topic != "first" {
		t.Fatalf("got session %+v", s)
	}
}

func TestActiveSession_FallsBackToFirst(t *testing.T) {
	c := newChat()
	c.ActiveSessionID = 999 // non-existent
	s := c.ActiveSession()
	if s.ID != 1 {
		t.Fatalf("expected fallback to first session, got %d", s.ID)
	}
	if c.ActiveSessionID != 1 {
		t.Fatalf("ActiveSessionID should be updated to 1")
	}
}

func TestActiveSession_CreatesDefaultWhenEmpty(t *testing.T) {
	c := &chat.Chat{Sessions: nil}
	s := c.ActiveSession()
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.ID != chat.DefaultSessionID || s.Topic != chat.DefaultSessionTopic {
		t.Fatalf("unexpected default: %+v", s)
	}
	if len(c.Sessions) != 1 {
		t.Fatalf("should have 1 session")
	}
	if c.NextSessionID != chat.DefaultNextSessionID {
		t.Fatalf("NextSessionID = %d", c.NextSessionID)
	}
}

// ===================== FindSession =====================

func TestFindSession_Found(t *testing.T) {
	c := newChat()
	s := c.FindSession(2)
	if s == nil || s.Topic != "second" {
		t.Fatalf("got %+v", s)
	}
}

func TestFindSession_NotFound(t *testing.T) {
	c := newChat()
	if c.FindSession(999) != nil {
		t.Fatal("expected nil")
	}
}

// ===================== RemoveSession =====================

func TestRemoveSession_Success(t *testing.T) {
	c := newChat()
	if !c.RemoveSession(2) {
		t.Fatal("expected true")
	}
	if len(c.Sessions) != 1 {
		t.Fatalf("len = %d", len(c.Sessions))
	}
	if c.FindSession(2) != nil {
		t.Fatal("session 2 should be gone")
	}
}

func TestRemoveSession_SwitchesActive(t *testing.T) {
	c := newChat()
	c.ActiveSessionID = 2
	c.RemoveSession(2)
	if c.ActiveSessionID != 1 {
		t.Fatalf("ActiveSessionID = %d, want 1", c.ActiveSessionID)
	}
}

func TestRemoveSession_LastSession(t *testing.T) {
	c := &chat.Chat{Sessions: []*chat.Session{{ID: 1}}}
	if c.RemoveSession(1) {
		t.Fatal("should not remove last session")
	}
}

func TestRemoveSession_NotFound(t *testing.T) {
	c := newChat()
	if c.RemoveSession(999) {
		t.Fatal("expected false")
	}
}

// ===================== AddSession =====================

func TestAddSession(t *testing.T) {
	c := newChat()
	s := c.AddSession("new topic")
	if s.ID != 3 {
		t.Errorf("ID = %d, want 3", s.ID)
	}
	if s.Topic != "new topic" {
		t.Errorf("Topic = %q", s.Topic)
	}
	if s.Model != "basic" { // inherits from active session (ID=1, model=basic)
		t.Errorf("Model = %q, want basic", s.Model)
	}
	if c.NextSessionID != 4 {
		t.Errorf("NextSessionID = %d", c.NextSessionID)
	}
	if len(c.Sessions) != 3 {
		t.Errorf("len = %d", len(c.Sessions))
	}
}

func TestAddSession_InheritsModel(t *testing.T) {
	c := newChat()
	c.ActiveSessionID = 2 // model = "fast"
	s := c.AddSession("test")
	if s.Model != "fast" {
		t.Errorf("Model = %q, want fast", s.Model)
	}
}

// ===================== FileStorage =====================

func TestFileStorage_SetAndGet(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStorage(dir)
	if err != nil {
		t.Fatal(err)
	}

	chat := &chat.Chat{
		ChatID: 42,
		Sessions: []*chat.Session{
			{ID: 1, Topic: "test", History: make([]*chat.ConversationEntry, 0), Model: "basic"},
		},
		ActiveSessionID: 1,
		NextSessionID:   2,
	}
	fs.Set(42, chat)

	got, ok := fs.Get(42)
	if !ok || got.ChatID != 42 {
		t.Fatalf("Get returned ok=%v, chat=%+v", ok, got)
	}
}

func TestFileStorage_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFileStorage(dir)

	_, ok := fs.Get(999)
	if ok {
		t.Fatal("expected not found")
	}
}

func TestFileStorage_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFileStorage(dir)

	chat := &chat.Chat{
		ChatID:          100,
		Title:           "saved chat",
		Sessions:        []*chat.Session{{ID: 1, Topic: "t", History: make([]*chat.ConversationEntry, 0), Model: "basic"}},
		ActiveSessionID: 1,
		NextSessionID:   2,
	}
	fs.Set(100, chat)
	if !fs.Save() {
		t.Fatal("Save failed")
	}

	// Verify file was created
	filePath := filepath.Join(dir, "100.chat")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Load into a new FileStorage
	fs2, _ := NewFileStorage(dir)
	got, ok := fs2.Get(100)
	if !ok {
		t.Fatal("not found after reload")
	}
	if got.Title != "saved chat" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.ActiveSession().Topic != "t" {
		t.Errorf("Topic = %q", got.ActiveSession().Topic)
	}
}

func TestFileStorage_MarkDirtyAndSave(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFileStorage(dir)

	chat := &chat.Chat{
		ChatID:          55,
		Sessions:        []*chat.Session{{ID: 1, Topic: "a", History: make([]*chat.ConversationEntry, 0), Model: "basic"}},
		ActiveSessionID: 1,
		NextSessionID:   2,
	}
	fs.Set(55, chat)
	fs.Save() // save initial state

	// Modify in-place and mark dirty
	chat.Title = "updated"
	fs.MarkDirty(55)
	fs.Save()

	// Reload
	fs2, _ := NewFileStorage(dir)
	got, _ := fs2.Get(55)
	if got.Title != "updated" {
		t.Errorf("Title = %q after MarkDirty+Save", got.Title)
	}
}

func TestFileStorage_SaveNothingDirty(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFileStorage(dir)
	if !fs.Save() {
		t.Fatal("Save with nothing dirty should return true")
	}
}

func TestFileStorage_CreatesDirIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir")
	_, err := NewFileStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir not created: %v", err)
	}
}

func TestFileStorage_LegacyNoSessions(t *testing.T) {
	// Simulate old format: a chat file with no sessions
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "77.chat"), []byte(`{"ChatID":77,"Title":"old"}`), 0644)

	fs, _ := NewFileStorage(dir)
	got, ok := fs.Get(77)
	if !ok {
		t.Fatal("not found")
	}
	// Should auto-create default session
	s := got.ActiveSession()
	if s.ID != chat.DefaultSessionID || s.Topic != chat.DefaultSessionTopic {
		t.Errorf("unexpected session: %+v", s)
	}
}

func TestFileStorage_All(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFileStorage(dir)

	for _, id := range []int64{10, 20, 30} {
		fs.Set(id, &chat.Chat{
			ChatID:          id,
			Sessions:        []*chat.Session{{ID: 1, Topic: "t", History: make([]*chat.ConversationEntry, 0), Model: "basic"}},
			ActiveSessionID: 1,
			NextSessionID:   2,
		})
	}
	fs.Save()

	// Create a fresh storage to ensure All reads from files
	fs2, _ := NewFileStorage(dir)
	all, err := fs2.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("All returned %d chats, want 3", len(all))
	}
	for _, id := range []int64{10, 20, 30} {
		if _, ok := all[id]; !ok {
			t.Errorf("chat %d missing", id)
		}
	}
}

// ===================== MemoryStorage =====================

func TestMemoryStorage_SetAndGet(t *testing.T) {
	ms := NewMemoryStorage()
	chat := &chat.Chat{ChatID: 42, Title: "mem"}
	ms.Set(42, chat)

	got, ok := ms.Get(42)
	if !ok || got.Title != "mem" {
		t.Fatalf("Get returned ok=%v, chat=%+v", ok, got)
	}
}

func TestMemoryStorage_GetNotFound(t *testing.T) {
	ms := NewMemoryStorage()
	_, ok := ms.Get(999)
	if ok {
		t.Fatal("expected not found")
	}
}

func TestMemoryStorage_MarkDirtyNoOp(t *testing.T) {
	ms := NewMemoryStorage()
	ms.MarkDirty(42) // should not panic
}

func TestMemoryStorage_SaveAlwaysTrue(t *testing.T) {
	ms := NewMemoryStorage()
	if !ms.Save() {
		t.Fatal("Save should always return true")
	}
}

func TestMemoryStorage_All(t *testing.T) {
	ms := NewMemoryStorage()
	ms.Set(1, &chat.Chat{ChatID: 1})
	ms.Set(2, &chat.Chat{ChatID: 2})

	all := ms.All()
	if len(all) != 2 {
		t.Fatalf("All returned %d chats, want 2", len(all))
	}
}

func TestMemoryStorage_Overwrite(t *testing.T) {
	ms := NewMemoryStorage()
	ms.Set(1, &chat.Chat{ChatID: 1, Title: "v1"})
	ms.Set(1, &chat.Chat{ChatID: 1, Title: "v2"})

	got, _ := ms.Get(1)
	if got.Title != "v2" {
		t.Errorf("Title = %q, want v2", got.Title)
	}
}

// ===================== Factory =====================

func TestNewStorage_File(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStorage("file", dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*FileStorage); !ok {
		t.Errorf("expected *FileStorage, got %T", s)
	}
}

func TestNewStorage_FileDefault(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStorage("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*FileStorage); !ok {
		t.Errorf("expected *FileStorage, got %T", s)
	}
}

func TestNewStorage_Memory(t *testing.T) {
	s, err := NewStorage("memory", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*MemoryStorage); !ok {
		t.Errorf("expected *MemoryStorage, got %T", s)
	}
}

func TestNewStorage_Unknown(t *testing.T) {
	_, err := NewStorage("redis", "")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}
