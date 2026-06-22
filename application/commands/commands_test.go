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
func (b *fakeBot) SendImageData(_ int64, _ []byte, _ string) error { return nil }
func (b *fakeBot) AudioUpload(_ int64, _ []byte) error             { return nil }
func (b *fakeBot) ReplyWithButtons(chatID int64, replyTo int, text string, _ bool, _ [][]sender.Button) error {
	b.sent = append(b.sent, sentMsg{chatID: chatID, replyTo: replyTo, text: text})
	return nil
}
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
	history := service.NewHistoryService()
	memory := service.NewMemoryService()
	mockClient := mock.NewClient()
	gptSvc := &service.GPTService{
		GptClient: mockClient,
		History:   history,
		Memory:    memory,
	}
	cmdSvc := &service.GPTCommandService{
		GptClient: mockClient,
	}
	chatSvc := service.NewChatService(
		storage.NewMemoryStorage(),
		service.ChatDefaults{LogDir: logDir},
		&fakeLog{},
	)
	registry := commands.NewRegistry()
	commands.RegisterAll(commands.Deps{
		Registry:      registry,
		CmdService:    cmdSvc,
		ChatService:   chatSvc,
		Notifier:      notifier,
		Auth:          auth,
		History:       history,
		Memory:        memory,
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
	for _, want := range []string{"help", "start", "clear", "model", "system", "markdown", "memory"} {
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

func TestCommandHelp_ListsCommands(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("help")
	ctx := makeCtx(1, 100, "/help") // admin user
	chat := newTestChat()
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "/start") {
		t.Error("help should contain /start")
	}
	if !strings.Contains(resp, "/clear") {
		t.Error("help should contain /clear")
	}
}

func TestCommandHelp_AdminSeesAdminCommands(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("help")
	ctx := makeCtx(1, 100, "/help") // 100 = admin
	chat := newTestChat()
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "администратора") {
		t.Error("admin should see admin commands section")
	}
}

func TestCommandHelp_NonAdminHidesAdminCommands(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("help")
	ctx := makeCtx(1, 200, "/help") // 200 = not admin
	chat := newTestChat()
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if strings.Contains(resp, "администратора") {
		t.Error("non-admin should not see admin section")
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
	ctx := makeCtx(1, 100, "/clear")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
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

// ======================== /model ========================

func TestCommandModel_ShowsCurrent(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("model")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/model")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Текущая модель") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandModel_SetValid(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("model")
	chat := newTestChat()

	if len(ai.Tiers) == 0 {
		t.Skip("no tiers defined")
	}
	target := ai.Tiers[len(ai.Tiers)-1] // pick a non-default tier
	ctx := makeCmdCtx(1, 100, "/model "+target.ID)
	responses := cmd.Execute(ctx, chat)

	if chat.ActiveSession().Model != target.ID {
		t.Errorf("model = %q, want %q", chat.ActiveSession().Model, target.ID)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	// The picker must carry one button per tier, with the chosen tier marked.
	rows := responses[0].Buttons
	if len(rows) != 1 || len(rows[0]) != len(ai.Tiers) {
		t.Fatalf("expected 1 row of %d buttons, got %v", len(ai.Tiers), rows)
	}
	var marked string
	for _, b := range rows[0] {
		if b.Data == "model:"+target.ID {
			marked = b.Text
		}
	}
	if !strings.HasPrefix(marked, "✅") {
		t.Errorf("selected tier button not marked: %q", marked)
	}
}

func TestCommandModel_ButtonCallback(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("model")
	chat := newTestChat()

	if len(ai.Tiers) < 2 {
		t.Skip("need at least 2 tiers")
	}
	target := ai.Tiers[len(ai.Tiers)-1]

	// Simulate a button tap: callback data "model:<id>" arrives as CommandArgs.
	ctx := makeCmdCtx(1, 100, "/model")
	ctx.CommandArgs = target.ID
	cmd.Execute(ctx, chat)

	if chat.ActiveSession().Model != target.ID {
		t.Errorf("model = %q, want %q", chat.ActiveSession().Model, target.ID)
	}
}

func TestCommandModel_InvalidName(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("model")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/model nonexistent_model_xyz")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "не найдена") {
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

// ======================== /markdown ========================

func TestCommandMarkdown_On(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("markdown")
	chat := newTestChat()
	chat.Settings.UseMarkdown = false
	ctx := makeCmdCtx(1, 100, "/markdown on")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "включен") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if !chat.Settings.UseMarkdown {
		t.Error("UseMarkdown should be true")
	}
}

func TestCommandMarkdown_Off(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("markdown")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/markdown off")
	cmd.Execute(ctx, chat)
	if chat.Settings.UseMarkdown {
		t.Error("UseMarkdown should be false")
	}
}

func TestCommandMarkdown_ShowStatus(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("markdown")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/markdown")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Markdown") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

// ======================== /memory ========================

func TestCommandMemory_Empty(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/memory")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "пуста") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandMemory_ShowFacts(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	chat.Memory = []string{"fact1", "fact2"}
	ctx := makeCmdCtx(1, 100, "/memory")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "fact1") || !strings.Contains(resp, "fact2") {
		t.Errorf("facts missing in: %q", resp)
	}
	if !strings.Contains(resp, "2 фактов") {
		t.Errorf("fact count missing in: %q", resp)
	}
}

func TestCommandMemory_Clear(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("memory")
	chat := newTestChat()
	chat.Memory = []string{"a", "b", "c"}
	ctx := makeCmdCtx(1, 100, "/memory clear")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "очищена") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if len(chat.Memory) != 0 {
		t.Error("memory should be empty")
	}
}

// ======================== /autoreply ========================

func TestCommandAutoReply_OnOff(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("autoreply")
	chat := newTestChat()
	chat.Settings.GroupAutoReply = false

	// "on" arg (typed "/autoreply on" or button tap "autoreply:on") enables it.
	onCtx := makeCmdCtx(1, 100, "/autoreply on")
	responses := cmd.Execute(onCtx, chat)
	if !chat.Settings.GroupAutoReply {
		t.Error("GroupAutoReply should be true after 'on'")
	}
	assertBoolButtons(t, responses, "autoreply", true)

	// "off" arg disables it.
	offCtx := makeCmdCtx(1, 100, "/autoreply off")
	responses = cmd.Execute(offCtx, chat)
	if chat.Settings.GroupAutoReply {
		t.Error("GroupAutoReply should be false after 'off'")
	}
	assertBoolButtons(t, responses, "autoreply", false)
}

func TestCommandAutoReply_ShowPanelNoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("autoreply")
	chat := newTestChat()
	chat.Settings.GroupAutoReply = true

	// No args: show the panel without changing state.
	responses := cmd.Execute(makeCmdCtx(1, 100, "/autoreply"), chat)
	if !chat.Settings.GroupAutoReply {
		t.Error("bare /autoreply must not change state")
	}
	assertBoolButtons(t, responses, "autoreply", true)
}

func TestCommandMarkdown_Buttons(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("markdown")
	chat := newTestChat()
	chat.Settings.UseMarkdown = false
	responses := cmd.Execute(makeCmdCtx(1, 100, "/markdown"), chat)
	assertBoolButtons(t, responses, "markdown", false)
}

// assertBoolButtons checks a boolean command's response: a single row with an
// on/off pair carrying "<cmd>:on" / "<cmd>:off", and the active state marked ✅.
func assertBoolButtons(t *testing.T, responses []sender.Response, cmd string, on bool) {
	t.Helper()
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	rows := responses[0].Buttons
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("expected one row of 2 buttons, got %v", rows)
	}
	var onBtn, offBtn sender.Button
	for _, b := range rows[0] {
		switch b.Data {
		case cmd + ":on":
			onBtn = b
		case cmd + ":off":
			offBtn = b
		}
	}
	if onBtn.Data == "" || offBtn.Data == "" {
		t.Fatalf("missing on/off button: %v", rows[0])
	}
	marked := offBtn
	if on {
		marked = onBtn
	}
	if !strings.HasPrefix(marked.Text, "✅") {
		t.Errorf("active state not marked: %q", marked.Text)
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
	if responses[0].ImageURL == "" {
		t.Error("admin should receive an image URL")
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

func TestCommandSessionNew(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("new")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/new my topic")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Создана") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if len(chat.Sessions) != 2 {
		t.Errorf("sessions count = %d, want 2", len(chat.Sessions))
	}
	if chat.Sessions[1].Topic != "my topic" {
		t.Errorf("topic = %q, want 'my topic'", chat.Sessions[1].Topic)
	}
	if chat.ActiveSessionID != chat.Sessions[1].ID {
		t.Error("new session should be active")
	}
}

func TestCommandSessionNew_DefaultTopic(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("new")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/new")
	cmd.Execute(ctx, chat)
	if chat.Sessions[1].Topic != "untitled" {
		t.Errorf("default topic = %q, want 'untitled'", chat.Sessions[1].Topic)
	}
}

func TestCommandSessionUse(t *testing.T) {
	deps, _ := buildDeps(t)
	chat := newTestChat()
	chat.AddSession("second")

	cmd, _ := deps.Registry.Get("use")
	ctx := makeCmdCtx(1, 100, "/use 2")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Переключено") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if chat.ActiveSessionID != 2 {
		t.Errorf("active = %d, want 2", chat.ActiveSessionID)
	}
}

func TestCommandSessionUse_NoArg(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("use")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/use")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Укажите ID") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandSessionUse_NotFound(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("use")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/use 99")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "не найдена") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandSessionRemove(t *testing.T) {
	deps, _ := buildDeps(t)
	chat := newTestChat()
	chat.AddSession("second")

	cmd, _ := deps.Registry.Get("remove")
	ctx := makeCmdCtx(1, 100, "/remove 2")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "удалена") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if len(chat.Sessions) != 1 {
		t.Errorf("sessions = %d, want 1", len(chat.Sessions))
	}
}

func TestCommandSessionRemove_LastSession(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("remove")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/remove 1")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "единственную") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandSessionUpdate(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("update")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/update 1 new topic name")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "переименована") {
		t.Errorf("unexpected reply: %q", resp)
	}
	if chat.Sessions[0].Topic != "new topic name" {
		t.Errorf("topic = %q", chat.Sessions[0].Topic)
	}
}

func TestCommandSessionUpdate_NoArgs(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("update")
	chat := newTestChat()
	ctx := makeCmdCtx(1, 100, "/update")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "Использование") {
		t.Errorf("unexpected reply: %q", resp)
	}
}

func TestCommandSessionCurrent(t *testing.T) {
	deps, _ := buildDeps(t)
	cmd, _ := deps.Registry.Get("current")
	chat := newTestChat()
	ctx := makeCtx(1, 100, "/current")
	resp := assertSingleReply(t, cmd.Execute(ctx, chat))
	if !strings.Contains(resp, "#1") {
		t.Errorf("should show session #1: %q", resp)
	}
	if !strings.Contains(resp, "default") {
		t.Errorf("should show topic: %q", resp)
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
