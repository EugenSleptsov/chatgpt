package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"testing"
)

// fakeProgress records progress/announce calls for assertions.
type fakeProgress struct {
	started   []string
	deleted   int
	announced []string
}

func (f *fakeProgress) StartProgress(_ int64, text string) func() {
	f.started = append(f.started, text)
	return func() { f.deleted++ }
}

func (f *fakeProgress) Announce(_ int64, text string) {
	f.announced = append(f.announced, text)
}

func TestStartProgress_NilReporterIsNoop(t *testing.T) {
	done := StartProgress(nil, 1, "working")
	if done == nil {
		t.Fatal("StartProgress(nil) returned nil func")
	}
	done() // must not panic
	Announce(nil, 1, "text")
}

func TestAnnounceTools_VerboseOff(t *testing.T) {
	p := &fakeProgress{}
	s := &GPTService{Progress: p}
	ch := &chatdomain.Chat{ChatID: 1}
	resp := &ai.Response{Output: []ai.ResponseOutputItem{{Type: "web_search_call"}}}

	s.announceTools(ch, resp, []ai.ToolCall{{Name: "save_note"}})
	if len(p.announced) != 0 {
		t.Fatalf("verbose off: expected no announcements, got %v", p.announced)
	}
}

func TestAnnounceTools_VerboseOn(t *testing.T) {
	p := &fakeProgress{}
	s := &GPTService{Progress: p}
	ch := &chatdomain.Chat{ChatID: 1}
	ch.Settings.Verbose = true
	resp := &ai.Response{Output: []ai.ResponseOutputItem{
		{Type: "web_search_call"},
		{Type: "message", Content: []ai.ResponseOutputContent{{Text: "hi"}}},
	}}

	s.announceTools(ch, resp, []ai.ToolCall{{Name: "save_note"}, {Name: "generate_voice"}})

	want := []string{"🔧 Вызван web_search", "🔧 Вызван save_note", "🔧 Вызван generate_voice"}
	if len(p.announced) != len(want) {
		t.Fatalf("announced = %v, want %v", p.announced, want)
	}
	for i := range want {
		if p.announced[i] != want[i] {
			t.Errorf("announced[%d] = %q, want %q", i, p.announced[i], want[i])
		}
	}
}

func TestAnnounceTools_NilProgressDoesNotPanic(t *testing.T) {
	s := &GPTService{}
	ch := &chatdomain.Chat{ChatID: 1}
	ch.Settings.Verbose = true
	resp := &ai.Response{Output: []ai.ResponseOutputItem{{Type: "image_generation_call"}}}
	s.announceTools(ch, resp, []ai.ToolCall{{Name: "save_note"}}) // must not panic
}
