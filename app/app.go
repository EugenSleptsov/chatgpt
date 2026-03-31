package app

import (
	"GPTBot/api/telegram"
	"GPTBot/application/commands"
	"GPTBot/application/service"
	conf "GPTBot/config"
	"GPTBot/domain/ai"
	"GPTBot/infrastructure/logger"
	"GPTBot/infrastructure/storage"
	"GPTBot/integration/ai/openai"
	"GPTBot/pipeline"
	"GPTBot/pipeline/decoder"
	"GPTBot/pipeline/executor"
	"GPTBot/pipeline/sender"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	numWorkers       = 10
	updateBufferSize = 100
)

// App is the top-level application object. It owns every dependency and
// orchestrates startup, worker pool, and graceful shutdown.
type App struct {
	bot         *telegram.Bot
	chatService *service.ChatService
	decoder     *decoder.Decoder
	sender      *sender.ResponseSender
	auth        *service.Auth
	notifier    *service.Notifier
}

// NewApp reads the config, creates all services and wires the handler pipeline.
func NewApp(configFile string) (*App, error) {
	logSystem := logger.NewSystem()

	config, err := conf.ReadConfig(configFile)
	if err != nil {
		return nil, err
	}

	bot, err := telegram.NewInstance(config.TelegramToken, config.CommandMenu, logSystem)
	if err != nil {
		return nil, err
	}

	notifier := service.NewNotifier(config.IgnoreReportIds, logSystem)
	if config.TelegramTokenLogBot != "" {
		adminLog, err := telegram.NewAdminLogger(config.TelegramTokenLogBot, config.AdminId)
		if err != nil {
			return nil, err
		}
		notifier.SetAdminLog(adminLog)
	}

	auth := service.NewAuth(config.AdminId, config.AuthorizedUserIds)
	configService := service.NewConfigService(config, configFile)

	historyService := service.NewHistoryService()
	memoryService := service.NewMemoryService()

	aiClient := openai.NewClient(config.GPTToken, logSystem)

	chatDefaults := service.ChatDefaults{
		MaxMessages:     config.MaxMessages,
		SummarizePrompt: config.SummarizePrompt,
		SystemPrompt:    config.DefaultSystemPrompt,
		LogDir:          config.LogDir,
		CostLimitUSD:    config.CostLimitUSD,
	}

	botStorage, err := storage.NewStorage(config.StorageType, config.DataDir, config.StorageDSN)
	if err != nil {
		return nil, err
	}
	chatService := service.NewChatService(botStorage, chatDefaults, logSystem)

	gptCommandService := &service.GPTCommandService{
		GptClient: aiClient,
		CostFn:    openai.CostForTokens,
		ImageCost: openai.ImageGenerationCost,
	}

	compactService := &service.CompactService{
		GptClient:       aiClient,
		CostFn:          openai.CostForTokens,
		ContextWindowFn: openai.ContextWindowForTier,
	}

	gptService := &service.GPTService{
		GptClient: aiClient,
		History:   historyService,
		Memory:    memoryService,
		Compact:   compactService,
		CostFn:    openai.CostForTokens,
		ImageCost: openai.ImageGenerationCost,
	}

	registry := commands.NewRegistry()
	commands.RegisterAll(registry, gptCommandService, chatService, notifier, auth, historyService, memoryService, configService, openai.ContextWindowForTier)

	return &App{
		bot:         bot,
		chatService: chatService,
		decoder:     buildDecoder(bot, bot.GetUsername(), aiClient, gptService, gptCommandService, historyService, notifier, auth, registry, config.DefaultAutoReplyPersona),
		sender:      buildResponseSender(bot, notifier),
		auth:        auth,
		notifier:    notifier,
	}, nil
}

// shutdownTimeout is the maximum time we wait for in-flight workers to drain.
//
// After this deadline the process exits regardless of pending work.
const shutdownTimeout = 30 * time.Second

// Run starts the update polling, worker pool and blocks until a shutdown
// signal is received or the update channel is closed.
//
// Updates are hash-partitioned by chat ID: every message from the same
// Telegram chat always lands on the same worker goroutine. This eliminates
// data races on *storage.Chat without per-chat mutexes.
//
// - First SIGINT/SIGTERM: stop accepting updates, drain workers with timeout
// - Second SIGINT: force-quit immediately (double Ctrl+C pattern)
// - Failsafe timer: exit after shutdownTimeout even if workers are stuck
func (a *App) Run() {
	updates := a.bot.GetUpdateChannel(60)

	// Per-worker channels — hash-partitioned by chatID.
	workerChans := make([]chan telegram.Update, numWorkers)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		workerChans[i] = make(chan telegram.Update, updateBufferSize)
		wg.Add(1)
		w := NewWorker(a.auth, a.bot, a.notifier, a.chatService, a.decoder, a.sender)
		go func(ch <-chan telegram.Update) {
			defer wg.Done()
			w.Start(ch)
		}(workerChans[i])
	}

	sigChan := make(chan os.Signal, 2) // buffered for 2: graceful + force
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				closeAll(workerChans)
				wg.Wait()
				a.chatService.Save()
				return
			}
			workerChans[partitionIndex(update, numWorkers)] <- update
		case sig := <-sigChan:
			log.Printf("Получен сигнал %v, начинаю graceful shutdown...", sig)
			closeAll(workerChans)

			// Drain workers with a timeout failsafe.
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			// Second signal = force quit (double Ctrl+C pattern).
			forceQuit := make(chan os.Signal, 1)
			signal.Notify(forceQuit, syscall.SIGINT, syscall.SIGTERM)

			select {
			case <-done:
				log.Println("Все воркеры завершены.")
			case <-time.After(shutdownTimeout):
				log.Printf("Таймаут %v: принудительное завершение (in-flight запросы потеряны).", shutdownTimeout)
			case sig2 := <-forceQuit:
				log.Printf("Повторный сигнал %v: принудительный выход.", sig2)
			}

			a.chatService.Save()
			log.Println("Данные сохранены. Выход.")
			return
		}
	}
}

// partitionIndex extracts the chat ID from an update and returns a
// stable worker index in [0, n). Updates without a message go to worker 0.
func partitionIndex(update telegram.Update, n int) int {
	if msg := update.Msg(); msg != nil && msg.Chat != nil {
		id := msg.Chat.ID
		if id < 0 {
			id = -id
		}
		return int(id % int64(n))
	}
	return 0
}

// closeAll closes every channel in the slice.
func closeAll(chans []chan telegram.Update) {
	for _, ch := range chans {
		close(ch)
	}
}

// --- wiring helpers (private to package app) ---

func buildDecoder(files pipeline.FileResolver, botUsername string, aiClient ai.Client, gpt *service.GPTService, cmds *service.GPTCommandService, history *service.HistoryService, notifier *service.Notifier, auth *service.Auth, registry *commands.Registry, defaultAutoReplyPersona string) *decoder.Decoder {
	d := decoder.NewDecoder()

	textExec := &executor.TextExecutor{
		BotUsername:             botUsername,
		GPT:                     gpt,
		Commands:                cmds,
		AIClient:                aiClient,
		History:                 history,
		Notifier:                notifier,
		Auth:                    auth,
		DefaultAutoReplyPersona: defaultAutoReplyPersona,
	}

	d.Register(&executor.CommandExecutor{Registry: registry, Auth: auth, Notifier: notifier})
	d.Register(&executor.VoiceExecutor{Files: files, AIClient: aiClient, Notifier: notifier, TextExecutor: textExec})
	d.Register(&executor.ImageExecutor{Files: files, BotUsername: botUsername, Commands: cmds, History: history, Notifier: notifier})
	d.Register(&executor.StickerExecutor{History: history, Notifier: notifier})
	d.Register(textExec) // catch-all — must be last

	return d
}

func buildResponseSender(bot sender.MessageSender, notifier *service.Notifier) *sender.ResponseSender {
	return &sender.ResponseSender{
		Bot:     bot,
		OnError: notifier.LogError,
	}
}
