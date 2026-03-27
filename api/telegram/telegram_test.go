package telegram

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ===================== Update.Msg() =====================

func TestUpdateMsg_Message(t *testing.T) {
	u := Update{Message: &tgbotapi.Message{MessageID: 1}}
	if u.Msg() == nil || u.Msg().MessageID != 1 {
		t.Fatal("expected Message")
	}
}

func TestUpdateMsg_EditedMessage(t *testing.T) {
	u := Update{EditedMessage: &tgbotapi.Message{MessageID: 2}}
	if u.Msg() == nil || u.Msg().MessageID != 2 {
		t.Fatal("expected EditedMessage")
	}
}

func TestUpdateMsg_ChannelPost(t *testing.T) {
	u := Update{ChannelPost: &tgbotapi.Message{MessageID: 3}}
	if u.Msg() == nil || u.Msg().MessageID != 3 {
		t.Fatal("expected ChannelPost")
	}
}

func TestUpdateMsg_EditedChannelPost(t *testing.T) {
	u := Update{EditedChannelPost: &tgbotapi.Message{MessageID: 4}}
	if u.Msg() == nil || u.Msg().MessageID != 4 {
		t.Fatal("expected EditedChannelPost")
	}
}

func TestUpdateMsg_Nil(t *testing.T) {
	u := Update{}
	if u.Msg() != nil {
		t.Fatal("expected nil")
	}
}

// ===================== Update.IsEdited() =====================

func TestIsEdited_EditedMessage(t *testing.T) {
	u := Update{EditedMessage: &tgbotapi.Message{}}
	if !u.IsEdited() {
		t.Fatal("expected true")
	}
}

func TestIsEdited_EditedChannelPost(t *testing.T) {
	u := Update{EditedChannelPost: &tgbotapi.Message{}}
	if !u.IsEdited() {
		t.Fatal("expected true")
	}
}

func TestIsEdited_NotEdited(t *testing.T) {
	u := Update{Message: &tgbotapi.Message{}}
	if u.IsEdited() {
		t.Fatal("expected false")
	}
}

// ===================== NewUpdateContext =====================

func newMsg(chatID int64, text string) *tgbotapi.Message {
	return &tgbotapi.Message{
		MessageID: 100,
		Text:      text,
		Chat:      &tgbotapi.Chat{ID: chatID, Title: "TestChat"},
		From:      &tgbotapi.User{ID: 42, FirstName: "John", LastName: "Doe", UserName: "johndoe"},
	}
}

func TestNewUpdateContext_Basic(t *testing.T) {
	u := Update{Message: newMsg(123, "hello")}
	ctx := NewUpdateContext(u)
	if ctx == nil {
		t.Fatal("ctx is nil")
	}
	if ctx.ChatID != 123 {
		t.Errorf("ChatID = %d", ctx.ChatID)
	}
	if ctx.MessageID != 100 {
		t.Errorf("MessageID = %d", ctx.MessageID)
	}
	if ctx.SenderID != 42 {
		t.Errorf("SenderID = %d", ctx.SenderID)
	}
	if ctx.SenderName != "John Doe" {
		t.Errorf("SenderName = %q", ctx.SenderName)
	}
	if ctx.Text != "hello" {
		t.Errorf("Text = %q", ctx.Text)
	}
	if ctx.IsGroup {
		t.Error("should not be group")
	}
	if ctx.IsEdited {
		t.Error("should not be edited")
	}
}

func TestNewUpdateContext_NilMsg(t *testing.T) {
	u := Update{}
	if ctx := NewUpdateContext(u); ctx != nil {
		t.Fatal("expected nil for empty update")
	}
}

func TestNewUpdateContext_Group(t *testing.T) {
	u := Update{Message: newMsg(-100123, "hi")}
	ctx := NewUpdateContext(u)
	if !ctx.IsGroup {
		t.Error("expected IsGroup=true for negative chatID")
	}
}

func TestNewUpdateContext_Caption(t *testing.T) {
	msg := newMsg(1, "")
	msg.Caption = "photo caption"
	u := Update{Message: msg}
	ctx := NewUpdateContext(u)
	if ctx.Text != "photo caption" {
		t.Errorf("Text = %q, want caption", ctx.Text)
	}
}

func TestNewUpdateContext_Photo(t *testing.T) {
	msg := newMsg(1, "")
	msg.Photo = []tgbotapi.PhotoSize{{FileID: "f1"}}
	u := Update{Message: msg}
	ctx := NewUpdateContext(u)
	if !ctx.IsPhoto {
		t.Error("expected IsPhoto")
	}
}

func TestNewUpdateContext_Voice(t *testing.T) {
	msg := newMsg(1, "")
	msg.Voice = &tgbotapi.Voice{FileID: "v1"}
	u := Update{Message: msg}
	ctx := NewUpdateContext(u)
	if !ctx.IsVoice {
		t.Error("expected IsVoice")
	}
}

func TestNewUpdateContext_Sticker(t *testing.T) {
	msg := newMsg(1, "")
	msg.Sticker = &tgbotapi.Sticker{FileID: "s1"}
	u := Update{Message: msg}
	ctx := NewUpdateContext(u)
	if !ctx.IsSticker {
		t.Error("expected IsSticker")
	}
}

func TestNewUpdateContext_Edited(t *testing.T) {
	u := Update{EditedMessage: newMsg(1, "edited")}
	ctx := NewUpdateContext(u)
	if !ctx.IsEdited {
		t.Error("expected IsEdited")
	}
}

// ===================== ChatTitle =====================

func TestChatTitle_PrivateChat(t *testing.T) {
	msg := newMsg(42, "")
	msg.Chat.FirstName = "Alice"
	msg.Chat.LastName = "B"
	msg.Chat.UserName = "alice_b"
	u := Update{Message: msg}
	ctx := NewUpdateContext(u)
	title := ctx.ChatTitle()
	if !strings.Contains(title, "Alice") || !strings.Contains(title, "alice_b") {
		t.Errorf("ChatTitle = %q", title)
	}
}

func TestChatTitle_GroupChat(t *testing.T) {
	msg := newMsg(-100999, "")
	msg.Chat.Title = "My Group"
	u := Update{Message: msg}
	ctx := NewUpdateContext(u)
	title := ctx.ChatTitle()
	if !strings.Contains(title, "My Group") || !strings.Contains(title, "-100999") {
		t.Errorf("ChatTitle = %q", title)
	}
}

// ===================== markdownToHTML =====================

func TestMarkdownToHTML_Bold(t *testing.T) {
	got := markdownToHTML("**bold**")
	if got != "<b>bold</b>" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_Italic_Underscore(t *testing.T) {
	got := markdownToHTML("_italic_")
	if got != "<i>italic</i>" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_Italic_Asterisk(t *testing.T) {
	got := markdownToHTML("*italic*")
	if got != "<i>italic</i>" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_Underline(t *testing.T) {
	got := markdownToHTML("__underline__")
	if got != "<u>underline</u>" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_InlineCode(t *testing.T) {
	got := markdownToHTML("`code`")
	if got != "<code>code</code>" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_CodeBlock(t *testing.T) {
	got := markdownToHTML("```go\nfmt.Println()\n```")
	if !strings.Contains(got, "<pre><code>") || !strings.Contains(got, "fmt.Println()") {
		t.Errorf("got %q", got)
	}
	// Language tag should be stripped
	if strings.Contains(got, "go\n") {
		t.Errorf("language tag not stripped: %q", got)
	}
}

func TestMarkdownToHTML_Link(t *testing.T) {
	got := markdownToHTML("[click](https://example.com)")
	if got != `<a href="https://example.com">click</a>` {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_HTMLEscape(t *testing.T) {
	got := markdownToHTML("<script>alert('xss')</script>")
	if strings.Contains(got, "<script>") {
		t.Errorf("HTML not escaped: %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped tags: %q", got)
	}
}

func TestMarkdownToHTML_PlainText(t *testing.T) {
	got := markdownToHTML("just text")
	if got != "just text" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_Nested(t *testing.T) {
	got := markdownToHTML("**bold _and italic_**")
	if !strings.Contains(got, "<b>") && !strings.Contains(got, "<i>") {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownToHTML_Empty(t *testing.T) {
	if got := markdownToHTML(""); got != "" {
		t.Errorf("got %q", got)
	}
}

// ===================== splitMessage =====================

func TestSplitMessage_Short(t *testing.T) {
	chunks := splitMessage("short")
	if len(chunks) != 1 || chunks[0] != "short" {
		t.Fatalf("chunks = %v", chunks)
	}
}

func TestSplitMessage_Long(t *testing.T) {
	// Build a message larger than maxMessageLen
	long := strings.Repeat("a", maxMessageLen+100)
	chunks := splitMessage(long)
	if len(chunks) < 2 {
		t.Fatalf("expected >=2 chunks, got %d", len(chunks))
	}
	// Each chunk should not exceed maxMessageLen (plus possible ``` wrapper)
	for i, ch := range chunks {
		if len([]rune(ch)) > maxMessageLen+10 { // small tolerance for ``` wrappers
			t.Errorf("chunk %d too long: %d runes", i, len([]rune(ch)))
		}
	}
}

func TestSplitMessage_PrefersNewlineSplit(t *testing.T) {
	// Build text with a newline near the end of the first chunk
	part1 := strings.Repeat("a", maxMessageLen-10)
	part2 := strings.Repeat("b", 20)
	text := part1 + "\n" + part2
	chunks := splitMessage(text)
	if len(chunks) < 2 {
		t.Fatalf("expected >=2 chunks")
	}
	// First chunk should end at the newline
	if strings.Contains(chunks[0], "b") {
		t.Error("first chunk should not contain part2")
	}
}

func TestSplitMessage_CodeBlockContinuity(t *testing.T) {
	// A code block that spans across a chunk boundary
	code := "```\n" + strings.Repeat("x\n", maxMessageLen/2) + "```"
	chunks := splitMessage(code)
	if len(chunks) < 2 {
		t.Fatalf("expected >=2 chunks, got %d", len(chunks))
	}
	// Each chunk should have balanced ``` (opened ones get closed, continued ones get opened)
	for i, ch := range chunks {
		fences := strings.Count(ch, "```")
		if fences%2 != 0 {
			t.Errorf("chunk %d has unbalanced fences (%d)", i, fences)
		}
	}
}

func TestSplitMessage_MaxChunks(t *testing.T) {
	huge := strings.Repeat("x", maxMessageLen*15)
	chunks := splitMessage(huge)
	if len(chunks) > maxChunks {
		t.Errorf("expected at most %d chunks, got %d", maxChunks, len(chunks))
	}
	last := chunks[len(chunks)-1]
	if !strings.Contains(last, "обрезано") {
		t.Error("last chunk should contain truncation indicator")
	}
}

// ===================== CommandDescriptions =====================

func TestCommandDescriptions_AllDefaultsExist(t *testing.T) {
	for _, cmd := range DefaultCommandList {
		if _, ok := CommandDescriptions[cmd]; !ok {
			t.Errorf("missing description for command %q", cmd)
		}
	}
}
