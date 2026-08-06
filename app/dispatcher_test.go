package app

import (
	"GPTBot/application/commands"
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/decoder"
	"GPTBot/pipeline/executor"
	"GPTBot/pipeline/sender"
	"sync"
	"testing"
	"time"
)

// gatedExecutor records which chats it handled and parks indefinitely inside
// one chosen chat, so a test can hold that chat's goroutine hostage the way a
// hung API call does in production.
type gatedExecutor struct {
	blockChatID int64
	entered     chan int64    // one value per parked call
	release     chan struct{} // closed to let parked calls finish

	mu      sync.Mutex
	handled []int64
}

func (e *gatedExecutor) Match(_ *pipeline.RequestContext) bool { return true }

func (e *gatedExecutor) Execute(ctx *pipeline.RequestContext, _ *chat.Chat) []sender.Response {
	if ctx.ChatID == e.blockChatID {
		// Non-blocking: once release is closed the queued jobs stream through
		// here, and nobody is reading entered any more.
		select {
		case e.entered <- ctx.ChatID:
		default:
		}
		<-e.release
	}
	e.mu.Lock()
	e.handled = append(e.handled, ctx.ChatID)
	e.mu.Unlock()
	return []sender.Response{{Text: "ok"}}
}

func (e *gatedExecutor) handledChats() []int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]int64(nil), e.handled...)
}

// gatedDispatcher wires a dispatcher whose workers all share one chat service,
// one bot and the given executor — the production shape, where per-chat
// goroutines run against shared services.
func gatedDispatcher(t *testing.T, exec *gatedExecutor) (*dispatcher, *fakeBot) {
	t.Helper()

	bot := &fakeBot{}
	log := &fakeLog{}
	auth := service.NewAuth(100, []int64{100, 200})
	notifier := &service.Notifier{Log: log}
	cs := fakeChatService()
	registry := commands.NewRegistry()

	dec := decoder.NewDecoder()
	dec.Register(&executor.CommandExecutor{Registry: registry, Auth: auth, Notifier: notifier})
	dec.Register(exec)

	resp := &sender.ResponseSender{Bot: bot}
	return newDispatcher(func() *Worker {
		return NewWorker(auth, bot, "test_bot", notifier, cs, dec, resp, nil)
	}), bot
}

func jobFor(chatID int64) Job {
	u := makeUpdate(chatID, 100, "hi")
	return Job{Update: &u}
}

// waitFor polls cond until it holds or the deadline passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// The whole point of the rewrite: a chat stuck in a long call must not stop
// any other chat. Under the old hash-partitioned pool these two chats could
// share a worker, and the stuck one would hold the other hostage.
func TestDispatcher_StuckChatDoesNotBlockOthers(t *testing.T) {
	exec := &gatedExecutor{blockChatID: 1, entered: make(chan int64, 4), release: make(chan struct{})}
	disp, _ := gatedDispatcher(t, exec)

	if !disp.Send(1, jobFor(1)) {
		t.Fatal("first job for chat 1 must be accepted")
	}
	<-exec.entered // chat 1 is now parked inside Execute

	if !disp.Send(2, jobFor(2)) {
		t.Fatal("job for chat 2 must be accepted while chat 1 is stuck")
	}
	waitFor(t, "chat 2 to be handled while chat 1 is stuck", func() bool {
		for _, id := range exec.handledChats() {
			if id == 2 {
				return true
			}
		}
		return false
	})

	close(exec.release)
	disp.Close()
	disp.Wait()
}

// Dispatch must never block, whatever a chat is doing — a blocking send is how
// one stuck chat used to silence the entire bot.
func TestDispatcher_SendDropsInsteadOfBlocking(t *testing.T) {
	exec := &gatedExecutor{blockChatID: 1, entered: make(chan int64, 4), release: make(chan struct{})}
	disp, _ := gatedDispatcher(t, exec)

	disp.Send(1, jobFor(1))
	<-exec.entered // one job in flight, mailbox empty

	// The mailbox holds exactly mailboxBuffer more; the next one is dropped.
	for i := 0; i < mailboxBuffer; i++ {
		if !disp.Send(1, jobFor(1)) {
			t.Fatalf("job %d should still fit in the mailbox", i)
		}
	}

	done := make(chan bool, 1)
	go func() { done <- disp.Send(1, jobFor(1)) }()
	select {
	case accepted := <-done:
		if accepted {
			t.Error("job past the mailbox capacity should be dropped, not accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("Send blocked on a full mailbox — dispatch must never block")
	}

	close(exec.release)
	disp.Close()
	disp.Wait()
}

// Per-chat ordering is what lets the whole codebase mutate *chat.Chat without
// mutexes, so it must survive the move to mailboxes.
func TestDispatcher_HandlesOneChatInOrder(t *testing.T) {
	exec := &gatedExecutor{entered: make(chan int64, 1), release: make(chan struct{})}
	disp, bot := gatedDispatcher(t, exec)

	for i := 0; i < 5; i++ {
		if !disp.Send(7, jobFor(7)) {
			t.Fatalf("job %d must be accepted", i)
		}
	}
	disp.Close()
	disp.Wait()

	if got := len(exec.handledChats()); got != 5 {
		t.Fatalf("handled %d jobs, want 5", got)
	}
	if got := len(bot.sent); got != 5 {
		t.Fatalf("sent %d replies, want 5 — concurrent handling would race here", got)
	}
}

// Close must not race with an in-flight Send: both take the same mutex, so a
// send can never land on an already-closed channel.
func TestDispatcher_ConcurrentSendAndClose(t *testing.T) {
	exec := &gatedExecutor{entered: make(chan int64, 1), release: make(chan struct{})}
	disp, _ := gatedDispatcher(t, exec)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			disp.Send(id, jobFor(id))
		}(int64(i%8) + 1)
	}
	go disp.Close()

	wg.Wait()
	disp.Close() // idempotent
	disp.Wait()
}
