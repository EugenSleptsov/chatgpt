package commands_test

import (
	"GPTBot/application/commands"
	"GPTBot/application/service"
	conf "GPTBot/config"
	"GPTBot/domain/ai"
	domain "GPTBot/domain/chat"
	"GPTBot/infrastructure/storage"
	"GPTBot/integration/ai/mock"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ======================== Fakes ========================

type fakeBot struct {
	sent []sentMsg
}

type sentMsg struct {
	chatID  int64
	replyTo int
	text    string
}

func (b *fakeBot) Reply(chatID int64, replyTo int, text string) {
	b.sent = append(b.sent, sentMsg{chatID: chatID, replyTo: replyTo, text: text})
}
func (b *fakeBot) ReplyMarkdown(chatID int64, replyTo int, text string, _ bool) {
	b.sent = append(b.sent, sentMsg{chatID: chatID, replyTo: replyTo, text: text})
}
func (b *fakeBot) Message(message string, chatID int64, _ bool)    {}
func (b *fakeBot) SendImage(_ int64, _ string, _ string) error     { return nil }
func (b *fakeBot) SendForceReply(_ int64, _ string) error          { return nil }
func (b *fakeBot) SendImageData(_ int64, _ []byte, _ string) error { return nil }
func (b *fakeBot) AudioUpload(_ int64, _ []byte) error             { return nil }
func (b *fakeBot) ReplyWithButtons(chatID int64, replyTo int, text string, _ bool, _ [][]sender.Button) (int, error) {
	b.sent = append(b.sent, sentMsg{chatID: chatID, replyTo: replyTo, text: text})
	return len(b.sent), nil // pseudo message ID
}
func (b *fakeBot) DeleteMessage(_ int64, _ int) error { return nil }
func (b *fakeBot) EditMessage(_ int64, _ int, _ string, _ bool, _ [][]sender.Button) error {
	return nil
}
func (b *fakeBot) AnswerCallback(_ string, _ string) error     { return nil }
func (b *fakeBot) GetFile(_ string) (pipeline.FileInfo, error) { return pipeline.FileInfo{}, nil }
func (b *fakeBot) FileURL(filePath string) string              { return "https://fake/" + filePath }
func (b *fakeBot) GetUsername() string                         { return "test_bot" }

type fakeLog struct{}

func (l *fakeLog) Log(_ string)                    {}
func (l *fakeLog) Logf(_ string, _ ...interface{}) {}
func (l *fakeLog) LogToFile(_ string, _ []string)  {}

// ======================== Helpers ========================

// testDeps holds all the components tests need access to, mirrors the fields
// that used to live in commands.Deps so call-sites (deps.Registry, deps.Config, …)
// remain unchanged.
type testDeps struct {
	Registry      *commands.Registry
	Config        *conf.Config
	ConfigService *service.ConfigService
	GPTService    *service.GPTService
	Notifier      *service.Notifier
	Auth          *service.Auth
}

func buildDeps(t *testing.T) (*testDeps, *fakeBot) {
	t.Helper()
	bot := &fakeBot{}
	logDir := t.TempDir()
	auth := service.NewAuth(100, []int64{100, 200})
	config := &conf.Config{DataDir: t.TempDir(), LogDir: logDir, SummarizePrompt: "summarize"}
	configService := service.NewConfigService(config, "")
	notifier := &service.Notifier{Log: &fakeLog{}}
	mockClient := mock.NewClient()
	gptSvc := &service.GPTService{GptClient: mockClient}
	chatSvc := service.NewChatService(
		storage.NewMemoryStorage(),
		service.ChatDefaults{LogDir: logDir},
		&fakeLog{},
	)
	registry := commands.NewRegistry()
	commands.RegisterAll(commands.Deps{
		Registry:      registry,
		CmdService:    gptSvc,
		ChatService:   chatSvc,
		Notifier:      notifier,
		Auth:          auth,
		ConfigService: configService,
	})
	return &testDeps{
		Registry:      registry,
		Config:        config,
		ConfigService: configService,
		GPTService:    gptSvc,
		Notifier:      notifier,
		Auth:          auth,
	}, bot
}

// setConfigPath creates a new ConfigService with the given path and propagates
// it to all admin commands that were already registered in the registry.
func setConfigPath(registry *commands.Registry, config *conf.Config, path string) {
	cs := service.NewConfigService(config, path)
	if cmd, err := registry.Get("adduser"); err == nil {
		cmd.(*commands.CommandAdminAddUser).ConfigService = cs
	}
	if cmd, err := registry.Get("removeuser"); err == nil {
		cmd.(*commands.CommandAdminRemoveUser).ConfigService = cs
	}
	if cmd, err := registry.Get("reload"); err == nil {
		cmd.(*commands.CommandAdminReload).ConfigService = cs
	}
}

func makeCtx(chatID, userID int64, text string) *pipeline.RequestContext {
	return &pipeline.RequestContext{
		ChatID:     chatID,
		SenderID:   userID,
		SenderName: "Test",
		Text:       text,
		ChatTitle:  "Test Chat",
	}
}

func makeCmdCtx(chatID, userID int64, fullCommand string) *pipeline.RequestContext {
	// Parse command name and arguments the same way tgbotapi does.
	parts := strings.SplitN(fullCommand, " ", 2)
	cmdName := strings.TrimPrefix(parts[0], "/")
	cmdArgs := ""
	if len(parts) > 1 {
		cmdArgs = parts[1]
	}
	return &pipeline.RequestContext{
		ChatID:      chatID,
		SenderID:    userID,
		SenderName:  "Test",
		Text:        fullCommand,
		ChatTitle:   "Test Chat",
		IsCommand:   true,
		CommandName: cmdName,
		CommandArgs: cmdArgs,
	}
}

func newTestChat() *domain.Chat {
	return &domain.Chat{
		ChatID: 1,
		Settings: domain.ChatSettings{
			UseMarkdown:     true,
			SummarizePrompt: "summarize",
		},
		Sessions: []*domain.Session{{
			ID:      1,
			Topic:   "default",
			History: make([]*domain.ConversationEntry, 0),
			Model:   ai.DefaultTierID,
		}},
		ActiveSessionID:  1,
		NextSessionID:    2,
		ImageGenNextTime: time.Now().Add(-time.Hour),
		Title:            "Test",
	}
}

func assertSingleReply(t *testing.T, responses []sender.Response) string {
	t.Helper()
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	return responses[0].Text
}

// hasButton reports whether any button across all rows has the given callback data.
func hasButton(rows [][]sender.Button, data string) bool {
	for _, row := range rows {
		for _, b := range row {
			if b.Data == data {
				return true
			}
		}
	}
	return false
}

// buttonText returns the text of the first button with the given callback data.
func buttonText(rows [][]sender.Button, data string) string {
	for _, row := range rows {
		for _, b := range row {
			if b.Data == data {
				return b.Text
			}
		}
	}
	return ""
}

// ======================== Settings / Menu hubs ========================

func TestCommandSettings_HubTogglesAndModel(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("settings")
	chat := newTestChat() // UseMarkdown = true, model = default

	// Admin hub shows toggles + admin-only rows.
	rows := cmd.Execute(makeCmdCtx(1, 100, "/settings"), chat)[0].Buttons
	if !strings.Contains(buttonText(rows, "settings:md"), "✅") {
		t.Errorf("markdown button should show ✅ when on: %q", buttonText(rows, "settings:md"))
	}
	if !hasButton(rows, "settings:ar") || !hasButton(rows, "settings:role") {
		t.Error("admin hub should include auto-reply and role rows")
	}

	// Toggle markdown off → re-rendered hub reflects it.
	rows = cmd.Execute(makeCmdCtx(1, 100, "/settings md"), chat)[0].Buttons
	if chat.Settings.UseMarkdown {
		t.Error("markdown should be toggled off")
	}
	if !strings.Contains(buttonText(rows, "settings:md"), "❌") {
		t.Errorf("markdown button should show ❌ after toggle: %q", buttonText(rows, "settings:md"))
	}

	// Model picker has a back row, and selecting a tier sets it.
	rows = cmd.Execute(makeCmdCtx(1, 100, "/settings model"), chat)[0].Buttons
	if !hasButton(rows, "settings:") {
		t.Error("model picker should have a back-to-hub button")
	}
	cmd.Execute(makeCmdCtx(1, 100, "/settings model:premium"), chat)
	if chat.ActiveSession().Model != "premium" {
		t.Errorf("model = %q, want premium", chat.ActiveSession().Model)
	}
}

func TestCommandSettings_NonAdminHidesAdminRows(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("settings")
	chat := newTestChat()

	rows := cmd.Execute(makeCmdCtx(1, 200, "/settings"), chat)[0].Buttons // 200 = non-admin
	if hasButton(rows, "settings:ar") || hasButton(rows, "settings:role") {
		t.Error("non-admin hub must not expose admin-only rows")
	}
	// Non-admin toggling auto-reply is a no-op (falls through to hub render).
	before := chat.Settings.GroupAutoReply
	cmd.Execute(makeCmdCtx(1, 200, "/settings ar"), chat)
	if chat.Settings.GroupAutoReply != before {
		t.Error("non-admin must not toggle auto-reply")
	}
}

func TestCommandSettings_VerboseToggle(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("settings")
	chat := newTestChat()

	// Admin hub shows the toggle; taps flip the flag both ways.
	rows := cmd.Execute(makeCmdCtx(1, 100, "/settings"), chat)[0].Buttons
	if !hasButton(rows, "settings:verbose") {
		t.Error("admin hub must show settings:verbose")
	}
	cmd.Execute(makeCmdCtx(1, 100, "/settings verbose"), chat)
	if !chat.Settings.Verbose {
		t.Error("verbose should be on after first toggle")
	}
	cmd.Execute(makeCmdCtx(1, 100, "/settings verbose"), chat)
	if chat.Settings.Verbose {
		t.Error("verbose should be off after second toggle")
	}

	// Non-admin: row hidden, toggle is a no-op.
	rows = cmd.Execute(makeCmdCtx(1, 200, "/settings"), chat)[0].Buttons
	if hasButton(rows, "settings:verbose") {
		t.Error("non-admin hub must hide settings:verbose")
	}
	cmd.Execute(makeCmdCtx(1, 200, "/settings verbose"), chat)
	if chat.Settings.Verbose {
		t.Error("non-admin must not toggle verbose")
	}
}

func TestCommandSettings_EditStartsForceReply(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("settings")
	chat := newTestChat()

	// Tapping "edit" for the system prompt arms pending input + force-reply.
	resp := cmd.Execute(makeCmdCtx(1, 100, "/settings system:edit"), chat)
	if len(resp) != 1 || !resp[0].ForceReply {
		t.Fatalf("expected a force-reply prompt, got %+v", resp)
	}
	if chat.PendingInput != "system" {
		t.Errorf("PendingInput = %q, want system", chat.PendingInput)
	}
}

func TestCommandMenu(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("menu")
	chat := newTestChat()

	rows := cmd.Execute(makeCmdCtx(1, 100, "/menu"), chat)[0].Buttons
	for _, want := range []string{"list:", "settings:", "menu:tools", "menu:info"} {
		if !hasButton(rows, want) {
			t.Errorf("main menu missing button %q", want)
		}
	}

	rows = cmd.Execute(makeCmdCtx(1, 100, "/menu info"), chat)[0].Buttons
	for _, want := range []string{"usage:", "context:", "history:", "menu:"} {
		if !hasButton(rows, want) {
			t.Errorf("info menu missing button %q", want)
		}
	}
}

// ======================== Registry ========================

func TestRegistry_AddAndGet(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, err := deps.Registry.Get("help")
	if err != nil {
		t.Fatalf("Get('help'): %v", err)
	}
	if cmd.Name() != "help" {
		t.Errorf("Name() = %q", cmd.Name())
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	deps, _ := buildDeps(t)
	_, err := deps.Registry.Get("nonexistent_cmd")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRegistry_AllReturnsAllCommands(t *testing.T) {
	deps, _ := buildDeps(t)
	all := deps.Registry.All()
	if len(all) == 0 {
		t.Fatal("expected at least one command")
	}
	names := make(map[string]bool)
	for _, cmd := range all {
		names[cmd.Name()] = true
	}
	for _, want := range []string{"help", "start", "clear", "system", "advisor", "settings", "list"} {
		if !names[want] {
			t.Errorf("command %q not registered", want)
		}
	}
}

func TestRegistry_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate Add")
		}
	}()
	reg := commands.NewRegistry()
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("help")
	reg.Add(cmd)
	reg.Add(cmd) // duplicate — should panic
}

// ======================== /start ========================

func TestCommandStart(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("start")
	ctx := makeCtx(1, 100, "/start")
	chat := newTestChat()
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Здравствуйте") {
		t.Errorf("unexpected start reply: %q", resp)
	}
}

// ======================== /help ========================

func TestCommandHelp_DefaultShowsCategories(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("help")
	rows := cmd.Execute(makeCmdCtx(1, 100, "/help"), newTestChat())[0].Buttons
	for _, data := range []string{"help:sessions", "help:memory", "help:tools", "help:settings", "help:chat"} {
		if !hasButton(rows, data) {
			t.Errorf("/help missing category %s", data)
		}
	}
}

func TestCommandHelp_CategoryLinksToHub(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("help")
	resp := cmd.Execute(makeCmdCtx(1, 100, "/help sessions"), newTestChat())[0]
	if !strings.Contains(resp.Text, "Сессии") || !hasButton(resp.Buttons, "list:") || !hasButton(resp.Buttons, "help:") {
		t.Fatalf("unexpected sessions help: %+v", resp)
	}
}

// ======================== /clear ========================

func TestCommandClear(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("clear")
	chat := newTestChat()
	chat.ActiveSession().History = []*domain.ConversationEntry{
		{Prompt: domain.Message{Role: "user", Content: "hi"}},
	}
	// Bare /clear shows a confirmation, does NOT clear yet.
	confirm := cmd.Execute(makeCmdCtx(1, 100, "/clear"), chat)
	if !hasButton(confirm[0].Buttons, "clear:yes") {
		t.Error("/clear should show a confirm button")
	}
	if len(chat.ActiveSession().History) != 1 {
		t.Error("history must not be cleared before confirmation")
	}

	// clear:yes performs the clear.
	resp := assertSingleReply(t, cmd.Execute(makeCmdCtx(1, 100, "/clear yes"), chat))
	if !strings.Contains(resp, "очищена") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if len(chat.ActiveSession().History) != 0 {
		t.Error("history should be empty after clear")
	}
}

// ======================== /rollback ========================

func TestCommandRollback_Default(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("rollback")
	chat := newTestChat()
	for i := 0; i < 5; i++ {
		chat.ActiveSession().History = append(chat.ActiveSession().History,
			&domain.ConversationEntry{Prompt: domain.Message{Role: "user", Content: "msg"}})
	}
	ctx := makeCmdCtx(1, 100, "/rollback")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "1") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if len(chat.ActiveSession().History) != 4 {
		t.Errorf("history length = %d, want 4", len(chat.ActiveSession().History))
	}
}

func TestCommandRollback_WithArg(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("rollback")
	chat := newTestChat()
	for i := 0; i < 5; i++ {
		chat.ActiveSession().History = append(chat.ActiveSession().History,
			&domain.ConversationEntry{Prompt: domain.Message{Role: "user", Content: "msg"}})
	}
	ctx := makeCmdCtx(1, 100, "/rollback 3")
	cmd.Execute(ctx, chat)
	if len(chat.ActiveSession().History) != 2 {
		t.Errorf("history length = %d, want 2", len(chat.ActiveSession().History))
	}
}

func TestCommandRollback_EmptyHistory(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("rollback")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/rollback")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "пуста") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /system ========================

func TestCommandSystem_ShowEmpty(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("system")
	chat := newTestChat()
	chat.ActiveSession().SystemPrompt = ""
	ctx := makeCmdCtx(1, 100, "/system")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "не установлено") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandSystem_ShowExisting(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("system")
	chat := newTestChat()
	chat.ActiveSession().SystemPrompt = "You are helpful"
	ctx := makeCmdCtx(1, 100, "/system")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "You are helpful") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandSystem_Set(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("system")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/system You are a pirate")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "установлено") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if chat.ActiveSession().SystemPrompt != "You are a pirate" {
		t.Errorf("system prompt = %q", chat.ActiveSession().SystemPrompt)
	}
}

func TestCommandSystem_TruncatesLong(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("system")
	chat := newTestChat()
	long := strings.Repeat("a", 2000)
	ctx := makeCmdCtx(1, 100, "/system "+long)
	cmd.Execute(ctx, chat)
	if len(chat.ActiveSession().SystemPrompt) != 1024 {
		t.Errorf("system prompt length = %d, want 1024", len(chat.ActiveSession().SystemPrompt))
	}
}

// ======================== settings: supermemory ========================

func TestCommandSettings_SupermemoryToggle(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("settings")
	chat := newTestChat()

	if chat.Settings.Supermemory {
		t.Fatal("supermemory must be off by default")
	}
	cmd.Execute(makeCmdCtx(1, 100, "/settings sm"), chat)
	if !chat.Settings.Supermemory {
		t.Error("first toggle must enable supermemory")
	}
	cmd.Execute(makeCmdCtx(1, 100, "/settings sm"), chat)
	if chat.Settings.Supermemory {
		t.Error("second toggle must disable supermemory")
	}
}

// ======================== /memory (supermemory hub) ========================

func TestCommandMemory_EmptyIndex(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()

	responses := cmd.Execute(makeCmdCtx(1, 100, "/memory"), chat)
	if !strings.Contains(responses[0].Text, "Узлов пока нет") {
		t.Errorf("unexpected empty view: %q", responses[0].Text)
	}
	if !hasButton(responses[0].Buttons, "memory:toggle") {
		t.Error("empty view must offer the enable/disable toggle")
	}
}

func TestCommandMemory_ToggleSetting(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()

	cmd.Execute(makeCmdCtx(1, 100, "/memory toggle"), chat)
	if !chat.Settings.Supermemory {
		t.Error("toggle must enable supermemory")
	}
	cmd.Execute(makeCmdCtx(1, 100, "/memory toggle"), chat)
	if chat.Settings.Supermemory {
		t.Error("second toggle must disable supermemory")
	}
}

func TestCommandMemory_NodeViewAndPin(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	n := chat.AddMemoryNode("обсуждение налогов", "детали саммари", 0, 4, 1)

	responses := cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/memory node:%d", n.ID)), chat)
	if !strings.Contains(responses[0].Text, "обсуждение налогов") || !strings.Contains(responses[0].Text, "детали саммари") {
		t.Errorf("node view missing content: %q", responses[0].Text)
	}
	if !hasButton(responses[0].Buttons, fmt.Sprintf("memory:src:%d", n.ID)) {
		t.Error("node with a source range must offer the source button")
	}

	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/memory pin:%d", n.ID)), chat)
	if !n.Pinned {
		t.Error("pin route must set Pinned")
	}
	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/memory pin:%d", n.ID)), chat)
	if n.Pinned {
		t.Error("pin route must toggle Pinned off")
	}
}

func TestCommandMemory_IndexShowsOnlyRoots(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	a := chat.AddMemoryNode("дочерний узел", "…", 0, 0, 1)
	parent := chat.AddMemoryNode("родительский узел", "…", 0, 0, 1)
	parent.Children = []int{a.ID}

	responses := cmd.Execute(makeCmdCtx(1, 100, "/memory"), chat)
	if hasButton(responses[0].Buttons, fmt.Sprintf("memory:node:%d", a.ID)) {
		t.Error("child node must not be listed in the root index")
	}
	if !hasButton(responses[0].Buttons, fmt.Sprintf("memory:node:%d", parent.ID)) {
		t.Error("parent node must be listed in the root index")
	}

	// Node view of the parent exposes the child for descent.
	responses = cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/memory node:%d", parent.ID)), chat)
	if !hasButton(responses[0].Buttons, fmt.Sprintf("memory:node:%d", a.ID)) {
		t.Error("parent view must list children buttons")
	}
}

func TestCommandMemory_DeleteWithConfirm(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	child := chat.AddMemoryNode("дочерний", "…", 0, 0, 1)
	parent := chat.AddMemoryNode("родитель", "…", 0, 0, 1)
	parent.Children = []int{child.ID}

	// First tap asks for confirmation, nothing deleted yet.
	responses := cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/memory del:%d", parent.ID)), chat)
	if !strings.Contains(responses[0].Text, "Удалить узел") {
		t.Errorf("expected confirmation prompt, got: %q", responses[0].Text)
	}
	if !hasButton(responses[0].Buttons, fmt.Sprintf("memory:del:yes:%d", parent.ID)) {
		t.Error("confirmation must offer the yes button")
	}
	if chat.FindMemoryNode(parent.ID) == nil {
		t.Fatal("node must not be deleted before confirmation")
	}

	// Confirmation deletes the parent; the child is promoted back to a root.
	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/memory del:yes:%d", parent.ID)), chat)
	if chat.FindMemoryNode(parent.ID) != nil {
		t.Error("node must be deleted after confirmation")
	}
	if chat.FindMemoryNode(child.ID) == nil {
		t.Error("children must survive parent deletion")
	}
	roots := chat.RootMemoryNodes()
	if len(roots) != 1 || roots[0].ID != child.ID {
		t.Errorf("child must become a root, got %+v", roots)
	}
}

func TestCommandMemory_DeleteSkipConfirm(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	chat.Settings.SkipDeleteConfirm = true
	n := chat.AddMemoryNode("узел", "…", 0, 0, 1)

	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/memory del:%d", n.ID)), chat)
	if chat.FindMemoryNode(n.ID) != nil {
		t.Error("SkipDeleteConfirm must delete immediately")
	}
}

func TestCommandClear_ForceSnapshotsSupermemory(t *testing.T) {
	// Wire a GPT service with a real archive + compact service (mock client) so
	// /clear can force-snapshot the unsaved conversation tail.
	mockClient := mock.NewClient()
	archive := storage.NewMemoryArchive()
	gptSvc := service.NewGPTService(mockClient, &service.CompactService{GptClient: mockClient, Archive: archive}, nil, 0, nil, archive)
	cmd := &commands.CommandClear{Commands: gptSvc}

	chat := newTestChat()
	chat.Settings.Supermemory = true
	session := chat.ActiveSession()
	service.AppendHistory(session, domain.Message{Role: "user", Content: "важный разговор"})
	service.AttachResponse(session, domain.Message{Role: "assistant", Content: "важный ответ"})

	cmd.Execute(makeCmdCtx(1, 100, "/clear yes"), chat)

	if len(session.History) != 0 {
		t.Error("history must be cleared")
	}
	if len(chat.MemoryNodes) != 1 {
		t.Fatalf("clear must force-snapshot the unsaved tail, nodes=%d", len(chat.MemoryNodes))
	}
	if chat.MemoryNodes[0].From != 0 || chat.MemoryNodes[0].To != 2 {
		t.Errorf("node range = [%d, %d), want [0, 2)", chat.MemoryNodes[0].From, chat.MemoryNodes[0].To)
	}
	if chat.ArchiveSnapshotTo != 2 {
		t.Errorf("snapshot pointer = %d, want 2", chat.ArchiveSnapshotTo)
	}
}

// ======================== /imagine ========================

func TestCommandImagine_NoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("imagine")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/imagine")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "укажите текст") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandImagine_Cooldown(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("imagine")
	chat := newTestChat()
	chat.ImageGenNextTime = time.Now().Add(time.Hour) // cooldown active
	ctx := makeCmdCtx(1, 200, "/imagine a cat")       // non-admin user
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "доступна в") {
		t.Errorf("expected cooldown message, got: %q", resp)
	}
}

func TestCommandImagine_AdminBypassesCooldown(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("imagine")
	chat := newTestChat()
	chat.ImageGenNextTime = time.Now().Add(time.Hour) // cooldown active
	ctx := makeCmdCtx(1, 100, "/imagine a cat")       // admin
	responses := cmd.Execute(ctx, chat)
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	// admin should bypass cooldown and get image response
	if len(responses[0].ImageData) == 0 {
		t.Error("admin should receive image data")
	}
}

// ======================== Session commands ========================

func TestCommandSessionList(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("list")
	chat := newTestChat()
	ctx := makeCtx(1, 100, "/list")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "#1") {
		t.Errorf("should list session #1: %q", resp)
	}
	if !strings.Contains(resp, "default") {
		t.Errorf("should show topic 'default': %q", resp)
	}
}

// addSessions appends n extra sessions (IDs 2..n+1) to force pagination.
func addSessions(chat *domain.Chat, n int) {
	for i := 0; i < n; i++ {
		id := i + 2
		chat.Sessions = append(chat.Sessions, &domain.Session{
			ID:    id,
			Topic: fmt.Sprintf("s%d", id),
			Model: ai.DefaultTierID,
		})
	}
}

func TestCommandSessionList_Pagination(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("list")
	chat := newTestChat()
	addSessions(chat, 7) // 8 sessions total → 2 pages of 6

	// Page 1: 6 session rows + 1 new-session row + 1 nav row + 1 menu row.
	responses := cmd.Execute(makeCmdCtx(1, 100, "/list"), chat)
	rows := responses[0].Buttons
	if len(rows) != 9 {
		t.Fatalf("page 1: expected 6 session rows + new + nav + menu, got %d rows", len(rows))
	}
	nav := rows[len(rows)-2] // nav is second-to-last; menu row is last.
	// First page: page indicator + forward arrow only (no back arrow).
	if len(nav) != 2 {
		t.Fatalf("page 1 nav: expected [indicator, ▶], got %v", nav)
	}
	if nav[len(nav)-1].Data != "list:1" {
		t.Errorf("forward button should go to page 1, got %q", nav[len(nav)-1].Data)
	}

	// Page 2 (via "list:1"): 2 remaining sessions + new-session row + nav + menu.
	responses = cmd.Execute(makeCmdCtx(1, 100, "/list 1"), chat)
	rows = responses[0].Buttons
	if len(rows) != 5 {
		t.Fatalf("page 2: expected 2 session rows + new + nav + menu, got %d rows", len(rows))
	}
	nav = rows[len(rows)-2]
	if nav[0].Data != "list:0" {
		t.Errorf("back button should go to page 0, got %q", nav[0].Data)
	}
}

func TestCommandSessionList_SelectButton(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("list")
	chat := newTestChat()
	addSessions(chat, 7)

	// Simulate tapping session #8's button: callback data "list:use:8".
	responses := cmd.Execute(makeCmdCtx(1, 100, "/list use:8"), chat)
	if chat.ActiveSessionID != 8 {
		t.Fatalf("active session = %d, want 8", chat.ActiveSessionID)
	}
	// The view must jump to the page holding #8 (page 1) and mark it active.
	if !strings.Contains(responses[0].Text, "Активная: #8") {
		t.Errorf("view should show active #8: %q", responses[0].Text)
	}
	var marked bool
	for _, row := range responses[0].Buttons {
		for _, b := range row {
			if b.Data == "list:use:8" && strings.HasPrefix(b.Text, "▶") {
				marked = true
			}
		}
	}
	if !marked {
		t.Error("session #8 button should be marked active")
	}
}

func TestCommandSessionList_NewButtonShowsModes(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("list")
	chat := newTestChat()
	responses := cmd.Execute(makeCmdCtx(1, 100, "/list new"), chat)
	for _, data := range []string{"new:ask:summary", "new:ask:copy", "new:ask:clean"} {
		if !hasButton(responses[0].Buttons, data) {
			t.Errorf("missing creation mode %s", data)
		}
	}
	if len(chat.Sessions) != 1 {
		t.Error("mode picker must not create a session")
	}
}

func TestCommandSessionList_RenameButtonStartsForceReply(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("list")
	chat := newTestChat()

	responses := cmd.Execute(makeCmdCtx(1, 100, "/list rename"), chat)
	if len(responses) != 1 || !responses[0].ForceReply {
		t.Fatalf("expected a force-reply prompt, got %+v", responses)
	}
	if chat.PendingInput != "rename" {
		t.Errorf("PendingInput = %q, want rename", chat.PendingInput)
	}
}

func TestCommandSessionNew(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("new")
	chat := newTestChat()
	responses := cmd.Execute(makeCmdCtx(1, 100, "/new my topic"), chat)
	if len(chat.Sessions) != 2 {
		t.Fatalf("sessions count = %d, want 2", len(chat.Sessions))
	}
	if chat.Sessions[1].Topic != "my topic" {
		t.Errorf("topic = %q, want 'my topic'", chat.Sessions[1].Topic)
	}
	if chat.ActiveSessionID != chat.Sessions[1].ID {
		t.Error("new session should be active")
	}
	// The command re-renders the session list view.
	if !strings.Contains(responses[0].Text, "my topic") {
		t.Errorf("expected refreshed list with new session: %q", responses[0].Text)
	}
}

func TestCommandSessionNew_EmptyShowsModes(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("new")
	chat := newTestChat()
	resp := cmd.Execute(makeCmdCtx(1, 100, "/new"), chat)
	if len(chat.Sessions) != 1 || !hasButton(resp[0].Buttons, "new:ask:summary") {
		t.Fatalf("empty /new should show mode picker: %+v", resp)
	}
}

func TestCommandSessionNew_CallbackArmsModeAndSource(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("new")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/new ask:copy")
	ctx.IsCallback = true
	resp := cmd.Execute(ctx, chat)
	if !resp[0].ForceReply || chat.PendingInput != "new" || chat.PendingInputArgs != "copy:1" {
		t.Fatalf("unexpected pending state: %+v %+v", chat, resp)
	}
}

func TestCommandSessionNew_SummarySeedsTransferContext(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("new")
	chat := newTestChat()
	chat.ActiveSession().History = []*domain.ConversationEntry{{Prompt: domain.Message{Role: "user", Content: "formula"}}}
	cmd.Execute(makeCmdCtx(1, 100, "/new summary:1\nphysics"), chat)
	created := chat.ActiveSession()
	if created.Topic != "physics" || created.TransferContext == "" || len(created.History) != 0 {
		t.Fatalf("summary transfer was not seeded separately: %+v", created)
	}
}

func TestCommandSessionNew_CopyIsIndependent(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("new")
	chat := newTestChat()
	chat.ActiveSession().SystemPrompt = "rules"
	chat.ActiveSession().History = []*domain.ConversationEntry{{Prompt: domain.Message{Role: "user", Content: "formula"}}}
	cmd.Execute(makeCmdCtx(1, 100, "/new copy:1\nphysics"), chat)
	copy := chat.ActiveSession()
	if copy.SystemPrompt != "rules" || len(copy.History) != 1 {
		t.Fatalf("copy lost context: %+v", copy)
	}
	copy.History[0].Prompt.Content = "changed"
	if chat.Sessions[0].History[0].Prompt.Content != "formula" {
		t.Error("session histories share pointers")
	}
}

func TestCommandSessionRename(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("rename")
	chat := newTestChat()
	responses := cmd.Execute(makeCmdCtx(1, 100, "/rename shiny name"), chat)
	if chat.ActiveSession().Topic != "shiny name" {
		t.Errorf("topic = %q, want 'shiny name'", chat.ActiveSession().Topic)
	}
	if !strings.Contains(responses[0].Text, "shiny name") {
		t.Errorf("expected refreshed list: %q", responses[0].Text)
	}

	// Empty input keeps the old name.
	cmd.Execute(makeCmdCtx(1, 100, "/rename"), chat)
	if chat.ActiveSession().Topic != "shiny name" {
		t.Error("empty rename must not change the topic")
	}
}

func TestCommandSessionRemove(t *testing.T) {
	deps, _ := buildDeps(t)
	chat := newTestChat()
	chat.AddSession("second")

	cmd, _ := deps.Registry.Get("remove")

	// /remove 2 shows a confirm; nothing deleted yet.
	confirm := cmd.Execute(makeCmdCtx(1, 100, "/remove 2"), chat)
	if !hasButton(confirm[0].Buttons, "remove:yes:2") {
		t.Error("/remove <id> should show a confirm button")
	}
	if len(chat.Sessions) != 2 {
		t.Error("session must not be removed before confirmation")
	}

	// remove:yes:2 performs the delete and re-renders the list.
	resp := cmd.Execute(makeCmdCtx(1, 100, "/remove yes:2"), chat)
	if !strings.Contains(resp[0].Text, "Сессии") {
		t.Errorf("expected refreshed list, got: %q", resp[0].Text)
	}
	if len(chat.Sessions) != 1 {
		t.Errorf("sessions = %d, want 1", len(chat.Sessions))
	}
}

func TestCommandSessionRemove_LastSession(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("remove")
	chat := newTestChat()
	// Confirming deletion of the only session must be refused.
	resp := assertSingleReply(t, cmd.Execute(makeCmdCtx(1, 100, "/remove yes:1"), chat))
	if !strings.Contains(resp, "единственную") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /advisor ========================

// addNotes seeds the chat with advisor topics/entries.
func addNotes(chat *domain.Chat, topic string, notes ...string) *domain.AdvisorTopic {
	var t *domain.AdvisorTopic
	for _, n := range notes {
		t, _ = chat.AddAdvisorNote(topic, n)
	}
	return t
}

func TestCommandAdvisor_EmptyState(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	responses := cmd.Execute(makeCmdCtx(1, 100, "/advisor"), chat)
	if !strings.Contains(responses[0].Text, "Заметок пока нет") {
		t.Errorf("unexpected empty-state text: %q", responses[0].Text)
	}
	if !hasButton(responses[0].Buttons, "menu:") {
		t.Error("empty state should link back to the menu")
	}
}

func TestCommandAdvisor_TopicListAndView(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	topic := addNotes(chat, "Налоги", "декларация до мая", "проверить вычеты")

	// Topic list shows the topic with its count and a button into it.
	responses := cmd.Execute(makeCmdCtx(1, 100, "/advisor"), chat)
	if !strings.Contains(responses[0].Text, "Налоги — 2") {
		t.Errorf("topic list should show count: %q", responses[0].Text)
	}
	data := fmt.Sprintf("advisor:topic:%d", topic.ID)
	if !hasButton(responses[0].Buttons, data) {
		t.Errorf("missing topic button %q", data)
	}

	// Topic view lists entries and offers entry/topic delete.
	responses = cmd.Execute(makeCmdCtx(1, 100, "/advisor topic:"+fmt.Sprint(topic.ID)), chat)
	if !strings.Contains(responses[0].Text, "декларация до мая") {
		t.Errorf("entries missing: %q", responses[0].Text)
	}
	if !hasButton(responses[0].Buttons, fmt.Sprintf("advisor:del:%d", topic.ID)) ||
		!hasButton(responses[0].Buttons, fmt.Sprintf("advisor:deltopic:%d", topic.ID)) {
		t.Error("topic view should offer entry delete and topic delete")
	}
}

func TestCommandAdvisor_RemoveEntry(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	topic := addNotes(chat, "ИКЕА", "полки", "стол")

	// Delete picker: one 🗑 button per entry.
	responses := cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor del:%d", topic.ID)), chat)
	rm := fmt.Sprintf("advisor:rm:%d:%d", topic.ID, topic.Entries[0].ID)
	if !hasButton(responses[0].Buttons, rm) {
		t.Fatalf("missing delete button %q", rm)
	}

	// Deleting one entry keeps the topic with the remaining entry.
	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor rm:%d:%d", topic.ID, topic.Entries[0].ID)), chat)
	if len(topic.Entries) != 1 || topic.Entries[0].Text != "стол" {
		t.Fatalf("unexpected entries after delete: %+v", topic.Entries)
	}

	// Deleting the last entry drops the topic and falls back to the list.
	responses = cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor rm:%d:%d", topic.ID, topic.Entries[0].ID)), chat)
	if len(chat.AdvisorTopics) != 0 {
		t.Error("topic should be removed with its last entry")
	}
	if !strings.Contains(responses[0].Text, "Заметок пока нет") {
		t.Errorf("expected empty-state view: %q", responses[0].Text)
	}
}

func TestCommandAdvisor_DeleteTopicConfirm(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	topic := addNotes(chat, "Налоги", "декларация")

	// First tap asks for confirmation.
	responses := cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor deltopic:%d", topic.ID)), chat)
	yes := fmt.Sprintf("advisor:deltopic:yes:%d", topic.ID)
	if !hasButton(responses[0].Buttons, yes) {
		t.Fatalf("expected confirm button %q", yes)
	}
	if len(chat.AdvisorTopics) != 1 {
		t.Fatal("topic must not be deleted before confirmation")
	}

	// Confirmation deletes.
	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor deltopic:yes:%d", topic.ID)), chat)
	if len(chat.AdvisorTopics) != 0 {
		t.Error("topic should be deleted after confirmation")
	}
}

func TestCommandAdvisor_DeleteTopicSkipConfirm(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	chat.Settings.SkipDeleteConfirm = true
	topic := addNotes(chat, "Налоги", "декларация")

	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor deltopic:%d", topic.ID)), chat)
	if len(chat.AdvisorTopics) != 0 {
		t.Error("SkipDeleteConfirm should delete immediately")
	}
}

func TestCommandAdvisor_Merge(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	src := addNotes(chat, "Покупки", "молоко")
	dst := addNotes(chat, "ИКЕА", "полки", "стол")

	// Source picker offers both topics.
	responses := cmd.Execute(makeCmdCtx(1, 100, "/advisor merge"), chat)
	if !hasButton(responses[0].Buttons, fmt.Sprintf("advisor:merge:%d", src.ID)) {
		t.Fatal("merge source picker missing topic button")
	}

	// Destination picker excludes the source.
	responses = cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor merge:%d", src.ID)), chat)
	if hasButton(responses[0].Buttons, fmt.Sprintf("advisor:merge:%d:%d", src.ID, src.ID)) {
		t.Error("destination picker must not offer the source itself")
	}
	if !hasButton(responses[0].Buttons, fmt.Sprintf("advisor:merge:%d:%d", src.ID, dst.ID)) {
		t.Fatal("destination picker missing target button")
	}

	// Performing the merge moves entries and drops the source.
	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor merge:%d:%d", src.ID, dst.ID)), chat)
	if len(chat.AdvisorTopics) != 1 {
		t.Fatalf("expected 1 topic after merge, got %d", len(chat.AdvisorTopics))
	}
	if len(dst.Entries) != 3 {
		t.Errorf("expected 3 entries in destination, got %d", len(dst.Entries))
	}
}

func TestCommandAdvisor_ReminderDone(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	topic := addNotes(chat, "Налоги", "написать адвокату", "декларация")
	eid := topic.Entries[0].ID

	responses := cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor done:%d:%d", topic.ID, eid)), chat)
	if !strings.Contains(responses[0].Text, "Готово") {
		t.Errorf("unexpected done text: %q", responses[0].Text)
	}
	if topic.FindEntry(eid) != nil {
		t.Error("done must delete the entry")
	}

	// Tapping done again on the same (stale) message is harmless.
	responses = cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor done:%d:%d", topic.ID, eid)), chat)
	if !strings.Contains(responses[0].Text, "уже удалена") {
		t.Errorf("stale done should report already deleted, got: %q", responses[0].Text)
	}
}

func TestCommandAdvisor_ReminderSnooze(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	topic := addNotes(chat, "Налоги", "написать адвокату")
	e := topic.Entries[0]

	// +1 hour.
	before := time.Now()
	responses := cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor snooze:%d:%d:1h", topic.ID, e.ID)), chat)
	if e.RemindAt == nil {
		t.Fatal("snooze must set RemindAt")
	}
	if e.RemindAt.Before(before.Add(59*time.Minute)) || e.RemindAt.After(before.Add(61*time.Minute)) {
		t.Errorf("1h snooze landed at %v", e.RemindAt)
	}
	if !strings.Contains(responses[0].Text, "Перенесено") {
		t.Errorf("unexpected snooze text: %q", responses[0].Text)
	}
	// Snoozed message must drop its controls — a live button row allowed
	// double taps; the reminder fires again later with fresh buttons.
	if len(responses[0].Buttons) != 0 {
		t.Error("snoozed message must not keep buttons")
	}

	// Tomorrow morning.
	cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor snooze:%d:%d:1d", topic.ID, e.ID)), chat)
	tomorrow := time.Now().AddDate(0, 0, 1)
	if e.RemindAt.Day() != tomorrow.Day() || e.RemindAt.Hour() != 9 {
		t.Errorf("1d snooze must land tomorrow at 9:00, got %v", e.RemindAt)
	}
}

func TestCommandAdvisor_TopicViewShowsReminder(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("advisor")
	chat := newTestChat()
	topic := addNotes(chat, "Налоги", "написать адвокату")
	at := time.Date(2026, 7, 10, 16, 0, 0, 0, time.Local)
	topic.Entries[0].RemindAt = &at

	responses := cmd.Execute(makeCmdCtx(1, 100, fmt.Sprintf("/advisor topic:%d", topic.ID)), chat)
	if !strings.Contains(responses[0].Text, "⏰ 10.07 16:00") {
		t.Errorf("topic view must show reminder time, got: %q", responses[0].Text)
	}
}

// ======================== /summarize_prompt ========================

func TestCommandSummarizePrompt_Show(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("summarize_prompt")
	chat := newTestChat()
	chat.Settings.SummarizePrompt = "my custom prompt"
	ctx := makeCmdCtx(1, 100, "/summarize_prompt")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "my custom prompt") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandSummarizePrompt_Set(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("summarize_prompt")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/summarize_prompt new prompt text")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "установлен") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if chat.Settings.SummarizePrompt != "new prompt text" {
		t.Errorf("prompt = %q", chat.Settings.SummarizePrompt)
	}
}

// ======================== /translate ========================

func TestCommandTranslate_NoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("translate")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/translate")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "укажите текст") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /enhance ========================

func TestCommandEnhance_NoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("enhance")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/enhance")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "укажите текст") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /grammar ========================

func TestCommandGrammar_NoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("grammar")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/grammar")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "укажите текст") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /tech_translate ========================

func TestCommandTechTranslate_NoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("tech_translate")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/tech_translate")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "укажите текст") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== Admin: /adduser /removeuser /reload ========================

func TestCommandAdminAddUser(t *testing.T) {
	deps, _ := buildDeps(t)
	// Write config to temp file for adduser to persist
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bot.yaml")
	if err := conf.UpdateConfig(cfgPath, deps.Config); err != nil {
		t.Fatal(err)
	}
	setConfigPath(deps.Registry, deps.Config, cfgPath)

	cmd, _ := deps.Registry.Get("adduser")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/adduser 300")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "добавлен") {
		t.Errorf("unexpected reply: %q", resp)
	}
	users := deps.Auth.GetAuthorizedUsers()
	found := false
	for _, id := range users {
		if id == 300 {
			found = true
		}
	}
	if !found {
		t.Error("user 300 not in authorized list")
	}
}

func TestCommandAdminAddUser_NoArg(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("adduser")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/adduser")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Укажите ID") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandAdminAddUser_Duplicate(t *testing.T) {
	deps, _ := buildDeps(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bot.yaml")
	_ = conf.UpdateConfig(cfgPath, deps.Config)
	setConfigPath(deps.Registry, deps.Config, cfgPath)

	cmd, _ := deps.Registry.Get("adduser")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/adduser 200") // 200 already authorized
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "уже добавлен") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandAdminAddUser_InvalidID(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("adduser")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/adduser abc")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Некорректный") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandAdminRemoveUser(t *testing.T) {
	deps, _ := buildDeps(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bot.yaml")
	_ = conf.UpdateConfig(cfgPath, deps.Config)
	setConfigPath(deps.Registry, deps.Config, cfgPath)

	cmd, _ := deps.Registry.Get("removeuser")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/removeuser 200")
	responses := cmd.Execute(ctx, chat)
	if len(responses) < 1 {
		t.Fatal("expected at least 1 response")
	}
	all := ""
	for _, r := range responses {
		all += r.Text + " "
	}
	if !strings.Contains(all, "удалён") {
		t.Errorf("unexpected reply: %q", all)
	}

	users := deps.Auth.GetAuthorizedUsers()
	for _, id := range users {
		if id == 200 {
			t.Error("user 200 should have been removed")
		}
	}
}

func TestCommandAdminRemoveUser_NoArg(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("removeuser")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/removeuser")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Укажите ID") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandAdminReload(t *testing.T) {
	deps, _ := buildDeps(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bot.yaml")
	cfg := &conf.Config{
		TelegramToken:     "new_tok",
		GPTToken:          "new_gpt",
		TimeoutValue:      99,
		AdminId:           100,
		AuthorizedUserIds: []int64{100, 200, 300},
		DataDir:           "d",
		LogDir:            "l",
	}
	if err := conf.UpdateConfig(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	setConfigPath(deps.Registry, deps.Config, cfgPath)

	cmd, _ := deps.Registry.Get("reload")
	chat := newTestChat()
	ctx := makeCtx(1, 100, "/reload")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Config updated") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if deps.Config.TimeoutValue != 99 {
		t.Errorf("TimeoutValue = %d, want 99", deps.Config.TimeoutValue)
	}
}

// ======================== /history ========================

func TestCommandHistory_Empty(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("history")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/history")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "пуста") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /analyze ========================

func TestCommandAnalyze_NoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("analyze")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/analyze")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "укажите") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /summarize (empty log) ========================

func TestCommandSummarize_EmptyLog(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("summarize")
	chat := newTestChat()
	// Log dir exists but no log file => ReadChatLog returns error
	ctx := makeCmdCtx(1, 100, "/summarize")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	// either "Произошла ошибка" or "пуста"
	if !strings.Contains(resp, "ошибка") && !strings.Contains(resp, "пуста") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== Helpers: all commands have Name/Description ========================

func TestAllCommands_Metadata(t *testing.T) {
	deps, _ := buildDeps(t)
	for _, cmd := range deps.Registry.All() {
		if cmd.Name() == "" {
			t.Error("command with empty name")
		}
		if cmd.Description() == "" {
			t.Errorf("command %q has empty description", cmd.Name())
		}
	}
}

// ======================== Helpers: write log file for summarize/analyze ========================

func TestCommandSummarize_WithLogFile(t *testing.T) {
	deps, _ := buildDeps(t)
	chat := newTestChat()
	chat.ChatID = 42

	// Create log file
	logFile := filepath.Join(deps.Config.LogDir, "42.log")
	lines := []string{"alice: hello", "bob: hi", "alice: how are you?"}
	if err := os.WriteFile(logFile, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd, _ := deps.Registry.Get("summarize")
	ctx := makeCmdCtx(42, 100, "/summarize 10")
	responses := cmd.Execute(ctx, chat)
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	// mock GPT returns "[mock] echo: ..." — just verify we got a non-empty response
	if responses[0].Text == "" {
		t.Error("expected non-empty summarize response")
	}
}

func TestCommandDream_DisabledSupermemory(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("dream")
	chat := newTestChat()

	responses := cmd.Execute(makeCmdCtx(1, 100, "/dream"), chat)
	if !strings.Contains(responses[0].Text, "выключена") {
		t.Errorf("dream on disabled supermemory: %q", responses[0].Text)
	}
}

func TestCommandDream_TooFewNodes(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("dream")
	chat := newTestChat()
	chat.Settings.Supermemory = true
	chat.AddMemoryNode("хук", "саммари", 0, 2, 1)

	responses := cmd.Execute(makeCmdCtx(1, 100, "/dream"), chat)
	if !strings.Contains(responses[0].Text, "проснулся") || !strings.Contains(responses[0].Text, "мало") {
		t.Errorf("dream with 1 node: %q", responses[0].Text)
	}
}

func TestCommandMemory_IndexOffersDream(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	chat.Settings.Supermemory = true
	chat.AddMemoryNode("хук", "саммари", 0, 2, 1)

	responses := cmd.Execute(makeCmdCtx(1, 100, "/memory"), chat)
	if !hasButton(responses[0].Buttons, "dream:") {
		t.Error("index view must offer the dream button when supermemory is on")
	}
}

func TestCommandMemory_AutoDreamToggle(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	chat.Settings.Supermemory = true

	cmd.Execute(makeCmdCtx(1, 100, "/memory autodream"), chat)
	if !chat.Settings.AutoDream {
		t.Error("autodream toggle must enable the setting")
	}
	cmd.Execute(makeCmdCtx(1, 100, "/memory autodream"), chat)
	if chat.Settings.AutoDream {
		t.Error("second toggle must disable the setting")
	}
}
