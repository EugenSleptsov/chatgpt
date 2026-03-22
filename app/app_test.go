package app

import (
	"GPTBot/api/telegram"
	"GPTBot/application/commands"
	"GPTBot/application/service"
	conf "GPTBot/config"
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/infrastructure/storage"
	"GPTBot/integration/ai/mock"
	"GPTBot/pipeline"
	"GPTBot/pipeline/decoder"
	"GPTBot/pipeline/executor"
	"GPTBot/pipeline/sender"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ===================== Fakes =====================

// fakeBot implements sender.MessageSender + pipeline.FileResolver without touching the network.
type fakeBot struct {
	sent []fakeSent
}

type fakeSent struct {
	chatID    int64
	replyTo   int
	text      string
	imageURL  string
	imageData []byte
	audio     []byte
	caption   string
}

func (b *fakeBot) Reply(chatID int64, replyTo int, text string) {
	b.sent = append(b.sent, fakeSent{chatID: chatID, replyTo: replyTo, text: text})
}
func (b *fakeBot) ReplyMarkdown(chatID int64, replyTo int, text string, _ bool) {
	b.sent = append(b.sent, fakeSent{chatID: chatID, replyTo: replyTo, text: text})
}
func (b *fakeBot) Message(message string, chatID int64, _ bool) {
	b.sent = append(b.sent, fakeSent{chatID: chatID, text: message})
}
func (b *fakeBot) SendImage(chatID int64, imageUrl string, caption string) error {
	b.sent = append(b.sent, fakeSent{chatID: chatID, imageURL: imageUrl, caption: caption})
	return nil
}
func (b *fakeBot) SendImageData(chatID int64, data []byte, caption string) error {
	b.sent = append(b.sent, fakeSent{chatID: chatID, imageData: data, caption: caption})
	return nil
}
func (b *fakeBot) AudioUpload(chatID int64, bytes []byte) error {
	b.sent = append(b.sent, fakeSent{chatID: chatID, audio: bytes})
	return nil
}
func (b *fakeBot) GetFile(_ string) (pipeline.FileInfo, error) {
	return pipeline.FileInfo{FilePath: "fake/path"}, nil
}
func (b *fakeBot) FileURL(filePath string) string { return "https://fake/" + filePath }
func (b *fakeBot) GetUsername() string            { return "test_bot" }

// fakeLog implements logger.Log + logger.FileLog
type fakeLog struct{}

func (l *fakeLog) Log(_ string)                    {}
func (l *fakeLog) Logf(_ string, _ ...interface{}) {}
func (l *fakeLog) LogToFile(_ string, _ []string)  {}

// fakeChatService creates a ChatService with in-memory storage for testing.
func fakeChatService() *service.ChatService {
	return service.NewChatService(
		storage.NewMemoryStorage(),
		service.ChatDefaults{MaxMessages: 50, LogDir: "_var/log"},
		&fakeLog{},
	)
}

// fakeExecutor always matches and returns canned responses.
type fakeExecutor struct {
	matchFn   func(ctx *pipeline.RequestContext) bool
	responses []sender.Response
}

func (e *fakeExecutor) Match(ctx *pipeline.RequestContext) bool {
	if e.matchFn != nil {
		return e.matchFn(ctx)
	}
	return true
}
func (e *fakeExecutor) Execute(_ *pipeline.RequestContext, _ *chat.Chat) []sender.Response {
	return e.responses
}

// --- Helpers ---

func makeUpdate(chatID int64, userID int64, text string) telegram.Update {
	return telegram.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Text:      text,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			From:      &tgbotapi.User{ID: userID, FirstName: "Test", UserName: "tester"},
		},
	}
}

func makeGroupUpdate(chatID int64, userID int64, text string) telegram.Update {
	return telegram.Update{
		Message: &tgbotapi.Message{
			MessageID: 2,
			Text:      text,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "group", Title: "TestGroup"},
			From:      &tgbotapi.User{ID: userID, FirstName: "User"},
		},
	}
}

func makeCommandUpdate(chatID int64, userID int64, command string) telegram.Update {
	return telegram.Update{
		Message: &tgbotapi.Message{
			MessageID: 3,
			Text:      "/" + command,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			From:      &tgbotapi.User{ID: userID, FirstName: "Test"},
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: len(command) + 1},
			},
		},
	}
}

func buildTestDeps(bot *fakeBot) (*commands.Registry, *service.Auth, *service.Notifier, *service.GPTService, ai.Client) {
	log := &fakeLog{}
	auth := service.NewAuth(100, []int64{100, 200})
	notifier := &service.Notifier{Log: log}
	history := service.NewHistoryService()
	memory := service.NewMemoryService()
	mockClient := mock.NewClient()
	gptSvc := &service.GPTService{GptClient: mockClient, History: history, Memory: memory}
	cmdSvc := &service.GPTCommandService{GptClient: mockClient}
	chatSvc := fakeChatService()
	registry := commands.NewRegistry()
	config := &conf.Config{MaxMessages: 50, DataDir: "_var/data", LogDir: "_var/log"}
	configService := service.NewConfigService(config, "")
	commands.RegisterAll(registry, cmdSvc, chatSvc, notifier, auth, history, memory, configService)
	return registry, auth, notifier, gptSvc, mockClient
}

// makeReqCtx creates a pipeline.RequestContext from a telegram.Update via the
// same conversion path the production code uses (worker.toRequestContext).
func makeReqCtx(update telegram.Update) *pipeline.RequestContext {
	return toRequestContext(telegram.NewUpdateContext(update))
}

// ===================== Decoder tests =====================

func TestDecoder_MatchesFirstExecutor(t *testing.T) {
	d := decoder.NewDecoder()
	e1 := &fakeExecutor{responses: []sender.Response{{Text: "first"}}}
	e2 := &fakeExecutor{responses: []sender.Response{{Text: "second"}}}
	d.Register(e1)
	d.Register(e2)

	ctx := makeReqCtx(makeUpdate(1, 1, "hi"))
	got := d.Decode(ctx)
	if got != e1 {
		t.Fatal("should match first executor")
	}
}

func TestDecoder_NilWhenNoMatch(t *testing.T) {
	d := decoder.NewDecoder()
	d.Register(&fakeExecutor{
		matchFn:   func(ctx *pipeline.RequestContext) bool { return ctx.IsCommand },
		responses: []sender.Response{{Text: "cmd"}},
	})

	ctx := makeReqCtx(makeUpdate(1, 1, "plain text"))
	if d.Decode(ctx) != nil {
		t.Fatal("expected nil for non-command")
	}
}

// ===================== CommandExecutor tests =====================

type stubCommand struct {
	name   string
	admin  bool
	result []sender.Response
}

func (c *stubCommand) Name() string        { return c.name }
func (c *stubCommand) Description() string { return "stub" }
func (c *stubCommand) IsAdmin() bool       { return c.admin }
func (c *stubCommand) Execute(_ *pipeline.RequestContext, _ *chat.Chat) []sender.Response {
	return c.result
}

func TestCommandExecutor_ExecutesCommand(t *testing.T) {
	reg := commands.NewRegistry()
	reg.Add(&stubCommand{name: "test", result: []sender.Response{{Text: "ok"}}})

	log := &fakeLog{}
	exec := &executor.CommandExecutor{
		Registry: reg,
		Auth:     service.NewAuth(100, nil),
		Notifier: &service.Notifier{Log: log},
	}

	ctx := makeReqCtx(makeCommandUpdate(1, 200, "test"))
	resp := exec.Execute(ctx, &chat.Chat{})
	if len(resp) != 1 || resp[0].Text != "ok" {
		t.Fatalf("got %+v", resp)
	}
}

func TestCommandExecutor_NotFound(t *testing.T) {
	reg := commands.NewRegistry()
	log := &fakeLog{}
	exec := &executor.CommandExecutor{
		Registry: reg,
		Auth:     service.NewAuth(100, nil),
		Notifier: &service.Notifier{Log: log},
	}

	ctx := makeReqCtx(makeCommandUpdate(1, 200, "nope"))
	resp := exec.Execute(ctx, &chat.Chat{})
	if resp != nil {
		t.Fatalf("expected nil for unknown command, got %+v", resp)
	}
}

func TestCommandExecutor_AdminOnly_Denied(t *testing.T) {
	reg := commands.NewRegistry()
	reg.Add(&stubCommand{name: "secret", admin: true, result: []sender.Response{{Text: "admin"}}})

	log := &fakeLog{}
	exec := &executor.CommandExecutor{
		Registry: reg,
		Auth:     service.NewAuth(100, nil),
		Notifier: &service.Notifier{Log: log},
	}

	ctx := makeReqCtx(makeCommandUpdate(1, 200, "secret"))
	resp := exec.Execute(ctx, &chat.Chat{})
	if resp != nil {
		t.Fatalf("non-admin should get nil, got %+v", resp)
	}
}

func TestCommandExecutor_AdminOnly_Allowed(t *testing.T) {
	reg := commands.NewRegistry()
	reg.Add(&stubCommand{name: "secret", admin: true, result: []sender.Response{{Text: "admin"}}})

	log := &fakeLog{}
	exec := &executor.CommandExecutor{
		Registry: reg,
		Auth:     service.NewAuth(100, nil),
		Notifier: &service.Notifier{Log: log},
	}

	ctx := makeReqCtx(makeCommandUpdate(1, 100, "secret"))
	resp := exec.Execute(ctx, &chat.Chat{})
	if len(resp) != 1 || resp[0].Text != "admin" {
		t.Fatalf("admin should get response, got %+v", resp)
	}
}

// ===================== ResponseSender tests =====================

func TestResponseSender_Text(t *testing.T) {
	bot := &fakeBot{}
	s := &sender.ResponseSender{Bot: bot}
	s.Send(42, 1, []sender.Response{{Text: "hello", Markdown: true}})
	if len(bot.sent) != 1 || bot.sent[0].text != "hello" {
		t.Fatalf("sent = %+v", bot.sent)
	}
}

func TestResponseSender_ImageURL(t *testing.T) {
	bot := &fakeBot{}
	s := &sender.ResponseSender{Bot: bot}
	s.Send(42, 1, []sender.Response{{ImageURL: "https://img.png", Caption: "cap"}})
	if len(bot.sent) != 1 || bot.sent[0].imageURL != "https://img.png" {
		t.Fatalf("sent = %+v", bot.sent)
	}
}

func TestResponseSender_ImageData(t *testing.T) {
	bot := &fakeBot{}
	s := &sender.ResponseSender{Bot: bot}
	s.Send(42, 1, []sender.Response{{ImageData: []byte("png"), Caption: "img"}})
	if len(bot.sent) != 1 || string(bot.sent[0].imageData) != "png" {
		t.Fatalf("sent = %+v", bot.sent)
	}
}

func TestResponseSender_Audio(t *testing.T) {
	bot := &fakeBot{}
	s := &sender.ResponseSender{Bot: bot}
	s.Send(42, 1, []sender.Response{{Audio: []byte("ogg")}})
	if len(bot.sent) != 1 || string(bot.sent[0].audio) != "ogg" {
		t.Fatalf("sent = %+v", bot.sent)
	}
}

func TestResponseSender_Multiple(t *testing.T) {
	bot := &fakeBot{}
	s := &sender.ResponseSender{Bot: bot}
	s.Send(42, 1, []sender.Response{
		{Text: "msg1"},
		{Text: "msg2"},
		{Audio: []byte("audio")},
	})
	if len(bot.sent) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(bot.sent))
	}
}

// ===================== Worker.ProcessUpdate tests =====================

func buildTestWorker(bot *fakeBot) (*Worker, *service.ChatService) {
	registry, auth, notifier, _, _ := buildTestDeps(bot)
	cs := fakeChatService()

	registry.Add(&stubCommand{name: "ping", result: []sender.Response{{Text: "pong"}}})

	dec := decoder.NewDecoder()
	dec.Register(&executor.CommandExecutor{Registry: registry, Auth: auth, Notifier: notifier})
	dec.Register(&fakeExecutor{responses: []sender.Response{{Text: "ai-reply", Markdown: true}}})

	sender := &sender.ResponseSender{Bot: bot}
	w := NewWorker(auth, bot, notifier, cs, dec, sender)
	return w, cs
}

func TestWorker_ProcessUpdate_Command(t *testing.T) {
	bot := &fakeBot{}
	w, _ := buildTestWorker(bot)
	w.ProcessUpdate(makeCommandUpdate(42, 100, "ping"))

	if len(bot.sent) != 1 || bot.sent[0].text != "pong" {
		t.Fatalf("expected pong, got %+v", bot.sent)
	}
}

func TestWorker_ProcessUpdate_TextMessage(t *testing.T) {
	bot := &fakeBot{}
	w, _ := buildTestWorker(bot)
	w.ProcessUpdate(makeUpdate(42, 100, "hello"))

	if len(bot.sent) != 1 || bot.sent[0].text != "ai-reply" {
		t.Fatalf("expected ai-reply, got %+v", bot.sent)
	}
}

func TestWorker_ProcessUpdate_Unauthorized(t *testing.T) {
	bot := &fakeBot{}
	w, _ := buildTestWorker(bot)
	w.ProcessUpdate(makeUpdate(42, 999, "hello"))

	if len(bot.sent) != 1 || !strings.Contains(bot.sent[0].text, "нет доступа") {
		t.Fatalf("expected access denied, got %+v", bot.sent)
	}
}

func TestWorker_ProcessUpdate_NilMsg(t *testing.T) {
	bot := &fakeBot{}
	w, _ := buildTestWorker(bot)
	w.ProcessUpdate(telegram.Update{})
	if len(bot.sent) != 0 {
		t.Fatalf("expected no messages, got %+v", bot.sent)
	}
}

func TestWorker_ProcessUpdate_CreatesChat(t *testing.T) {
	bot := &fakeBot{}
	w, cs := buildTestWorker(bot)
	w.ProcessUpdate(makeUpdate(42, 100, "hi"))

	// Verify the chat was persisted by retrieving it through the service.
	ctx := makeReqCtx(makeUpdate(42, 100, ""))
	c := cs.GetOrCreateChat(ctx)
	if c.ChatID != 42 {
		t.Fatal("chat should be created in ChatService")
	}
}

func TestWorker_ProcessUpdate_SavesAfter(t *testing.T) {
	bot := &fakeBot{}
	w, _ := buildTestWorker(bot)

	ch := make(chan telegram.Update, 1)
	ch <- makeUpdate(42, 100, "hi")
	close(ch)

	// Save is called by Start after every update; with MemoryStorage it is
	// a no-op, so we just verify that Start completes without panic.
	w.Start(ch)
}

// ===================== buildDecoder / buildResponseSender =====================

func TestBuildDecoder_ReturnsDecoder(t *testing.T) {
	bot := &fakeBot{}
	registry, auth, notifier, gptSvc, aiClient := buildTestDeps(bot)
	cmdSvc := &service.GPTCommandService{GptClient: aiClient}
	d := buildDecoder(bot, bot.GetUsername(), aiClient, gptSvc, cmdSvc, gptSvc.History, notifier, auth, registry, "")
	if d == nil {
		t.Fatal("buildDecoder returned nil")
	}
}

func TestBuildResponseSender(t *testing.T) {
	bot := &fakeBot{}
	_, _, notifier, _, _ := buildTestDeps(bot)
	s := buildResponseSender(bot, notifier)
	if s == nil {
		t.Fatal("buildResponseSender returned nil")
	}
	s.Send(1, 1, []sender.Response{{Text: "test"}})
	if len(bot.sent) != 1 {
		t.Fatalf("expected 1 message, got %d", len(bot.sent))
	}
}

// ===================== Integration: Worker through channel =====================

func TestWorker_StartProcessesChannel(t *testing.T) {
	bot := &fakeBot{}
	w, _ := buildTestWorker(bot)

	ch := make(chan telegram.Update, 3)
	ch <- makeUpdate(1, 100, "msg1")
	ch <- makeUpdate(1, 100, "msg2")
	ch <- makeCommandUpdate(1, 100, "ping")
	close(ch)

	w.Start(ch)
	if len(bot.sent) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(bot.sent), bot.sent)
	}
	if bot.sent[2].text != "pong" {
		t.Errorf("last message should be pong, got %q", bot.sent[2].text)
	}
}

// ===================== TextHandler with mock GPT =====================

func TestTextExecutor_PrivateChat(t *testing.T) {
	bot := &fakeBot{}
	_, auth, notifier, gptSvc, aiClient := buildTestDeps(bot)

	cmdSvc := &service.GPTCommandService{GptClient: aiClient}
	exec := &executor.TextExecutor{
		BotUsername: "test_bot",
		GPT:         gptSvc,
		Commands:    cmdSvc,
		AIClient:    aiClient,
		History:     gptSvc.History,
		Notifier:    notifier,
		Auth:        auth,
	}
	ctx := makeReqCtx(makeUpdate(1, 100, "hello"))
	chat := &chat.Chat{
		ChatID:   1,
		Settings: chat.ChatSettings{MaxMessages: 10, UseMarkdown: true},
		Sessions: []*chat.Session{{
			ID: 1, Topic: "test", Model: "basic",
			History: make([]*chat.ConversationEntry, 0),
		}},
		ActiveSessionID: 1,
		NextSessionID:   2,
	}

	resp := exec.Execute(ctx, chat)
	if len(resp) == 0 {
		t.Fatal("expected at least one response")
	}
	if !strings.Contains(resp[0].Text, "mock") {
		t.Errorf("expected mock response, got %q", resp[0].Text)
	}
	if len(chat.ActiveSession().History) == 0 {
		t.Error("history should have at least one entry")
	}
}
