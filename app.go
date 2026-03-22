package main

import (
	"GPTBot/api/gpt/openai"
	"GPTBot/api/logger"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	conf "GPTBot/config"
	"GPTBot/handler"
	"GPTBot/handler/execute"
	"GPTBot/handler/normalize"
	"GPTBot/manager"
	"GPTBot/service"
	"GPTBot/storage"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	numWorkers       = 10
	updateBufferSize = 100
)

// App is the top-level application object. It owns every dependency and
// orchestrates startup, worker pool, and graceful shutdown.
type App struct {
	log         logger.Logger
	bot         *telegram.Bot
	chatManager manager.ChatManager
	router      *handler.Router
	dispatcher  *handler.Dispatcher
	sender      *handler.ResponseSender
	deps        *commands.Deps
}

// NewApp reads the config, creates all services and wires the handler pipeline.
func NewApp(configFile string) (*App, error) {
	logSystem := logger.NewSystem()

	config, err := conf.ReadConfig(configFile)
	if err != nil {
		return nil, err
	}

	bot, err := telegram.NewInstance(config, logSystem)
	if err != nil {
		return nil, err
	}

	notifier := &service.Notifier{
		Log:             logSystem,
		IgnoreReportIDs: config.IgnoreReportIds,
	}
	if config.TelegramTokenLogBot != "" {
		adminLog, err := telegram.NewAdminLogger(config.TelegramTokenLogBot, config.AdminId)
		if err != nil {
			return nil, err
		}
		notifier.AdminLog = adminLog
	}

	auth := service.NewAuth(config.AdminId, config.AuthorizedUserIds)

	gptService := &service.GPTService{
		GptClient: openai.NewClient(config.GPTToken, logSystem),
		LogDir:    config.LogDir,
	}

	deps := &commands.Deps{
		Bot:        bot,
		Config:     config,
		ConfigPath: configFile,
		Registry:   commands.NewRegistry(),
		GPTService: gptService,
		Notifier:   notifier,
		Auth:       auth,
	}
	commands.RegisterAll(deps)

	botStorage, err := storage.NewFileStorage(config.DataDir)
	if err != nil {
		return nil, err
	}

	return &App{
		log:         logSystem,
		bot:         bot,
		chatManager: manager.NewTelegramChatManager(botStorage, config, logSystem),
		router:      buildRouter(deps),
		dispatcher:  buildDispatcher(deps),
		sender:      buildResponseSender(deps),
		deps:        deps,
	}, nil
}

// Run starts the update polling, worker pool and blocks until a shutdown
// signal is received or the update channel is closed.
func (a *App) Run() {
	updates := a.bot.GetUpdateChannel(a.deps.Config.TimeoutValue)
	updateChan := make(chan telegram.Update, updateBufferSize)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		w := NewWorker(a.deps, a.chatManager, a.router, a.dispatcher, a.sender)
		go func() {
			defer wg.Done()
			w.Start(updateChan)
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				close(updateChan)
				wg.Wait()
				a.chatManager.Save()
				return
			}
			updateChan <- update
		case sig := <-sigChan:
			log.Printf("Получен сигнал %v, завершение...", sig)
			close(updateChan)
			wg.Wait()
			a.chatManager.Save()
			log.Println("Данные сохранены. Выход.")
			return
		}
	}
}

// --- wiring helpers (private to package main) ---

func buildRouter(deps *commands.Deps) *handler.Router {
	r := handler.NewRouter()
	r.RegisterAll(normalize.AllHandlers(deps))
	return r
}

func buildDispatcher(deps *commands.Deps) *handler.Dispatcher {
	cmdBranch := &commands.Branch{
		Registry: deps.Registry,
		Auth:     deps.Auth,
		Notifier: deps.Notifier,
	}

	pipeline := handler.NewPipeline(&handler.IntentResolver{})
	pipeline.RegisterAll(execute.AllExecutors(deps))

	return handler.NewDispatcher(cmdBranch, pipeline)
}

func buildResponseSender(deps *commands.Deps) *handler.ResponseSender {
	return &handler.ResponseSender{
		Bot:     deps.Bot,
		OnError: deps.Notifier.LogError,
	}
}
