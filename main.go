package main

import (
	"GPTBot/api/gpt/openai"
	"GPTBot/api/logger"
	"GPTBot/api/telegram"
	"GPTBot/api/telegram/adminlog"
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
	configFile       = "bot.yaml"
)

func main() {
	logSystem := logger.NewSystem()

	config, err := conf.ReadConfig(configFile)
	logSystem.LogFatal(err)

	telegramBot, err := telegram.NewInstance(config, logSystem)
	logSystem.LogFatal(err)

	// Admin notification bot (optional)
	notifier := &service.Notifier{
		Log:             logSystem,
		IgnoreReportIDs: config.IgnoreReportIds,
	}
	if config.TelegramTokenLogBot != "" {
		adminLog, err := adminlog.NewTelegramAdminLogger(config.TelegramTokenLogBot, config.AdminId)
		logSystem.LogFatal(err)
		notifier.AdminLog = adminLog
	}

	auth := service.NewAuth(config.AdminId, config.AuthorizedUserIds)

	gptService := &service.GPTService{
		GptClient: openai.NewClient(config.GPTToken, logSystem),
		LogDir:    config.LogDir,
	}

	deps := &commands.Deps{
		Bot:        telegramBot,
		Config:     config,
		ConfigPath: configFile,
		Registry:   commands.NewCommandFactory(),
		GPTService: gptService,
		Notifier:   notifier,
		Auth:       auth,
	}
	commands.RegisterAll(deps)

	botStorage, err := storage.NewFileStorage(config.DataDir)
	logSystem.LogFatal(err)

	chatManager := manager.NewTelegramChatManager(botStorage, config, logSystem)
	router := buildRouter(deps)
	dispatcher := buildDispatcher(deps)
	sender := buildResponseSender(deps)

	startWorkers(deps, chatManager, router, dispatcher, sender, telegramBot.GetUpdateChannel(config.TimeoutValue))
}

// buildRouter wires all normalizer handlers in priority order.
func buildRouter(deps *commands.Deps) *handler.Router {
	r := handler.NewRouter()
	r.Register(&normalize.CommandHandler{})
	r.Register(&normalize.VoiceHandler{Deps: deps})
	r.Register(&normalize.ImageHandler{Deps: deps})
	r.Register(&normalize.StickerHandler{Deps: deps})
	r.Register(&normalize.MessageHandler{Deps: deps}) // catch-all
	return r
}

// buildDispatcher wires the command branch and the conversational pipeline.
func buildDispatcher(deps *commands.Deps) *handler.Dispatcher {
	// Command branch
	cmdBranch := &commands.Branch{
		Registry: deps.Registry,
		Auth:     deps.Auth,
		Notifier: deps.Notifier,
	}

	// Conversational branch (pipeline)
	pipeline := handler.NewPipeline(&handler.IntentResolver{})
	pipeline.RegisterExecutor(handler.IntentChat, &execute.ChatExecutor{Deps: deps})
	pipeline.RegisterExecutor(handler.IntentGroupReply, &execute.GroupReplyExecutor{Deps: deps})
	pipeline.RegisterExecutor(handler.IntentGroupAutoReply, &execute.GroupAutoReplyExecutor{Deps: deps})
	pipeline.RegisterExecutor(handler.IntentAnalyzeImage, &execute.ImageAnalysisExecutor{Deps: deps})
	pipeline.RegisterExecutor(handler.IntentEchoTranscription, &execute.EchoTranscriptionExecutor{Deps: deps})

	return handler.NewDispatcher(cmdBranch, pipeline)
}

// buildResponseSender creates the response delivery component.
func buildResponseSender(deps *commands.Deps) *handler.ResponseSender {
	return &handler.ResponseSender{
		Bot:     deps.Bot,
		OnError: deps.Notifier.LogError,
	}
}

func startWorkers(
	deps *commands.Deps,
	chatManager manager.ChatManager,
	router *handler.Router,
	dispatcher *handler.Dispatcher,
	sender *handler.ResponseSender,
	telegramUpdates telegram.UpdatesChannel,
) {
	updateChan := make(chan telegram.Update, updateBufferSize)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		worker := NewWorker(deps, chatManager, router, dispatcher, sender)
		go func() {
			defer wg.Done()
			worker.Start(updateChan)
		}()
	}

	// Listen for OS signals to trigger graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case update, ok := <-telegramUpdates:
			if !ok {
				close(updateChan)
				wg.Wait()
				chatManager.Save()
				return
			}
			updateChan <- update
		case sig := <-sigChan:
			log.Printf("Получен сигнал %v, завершение...", sig)
			close(updateChan)
			wg.Wait()
			chatManager.Save()
			log.Println("Данные сохранены. Выход.")
			return
		}
	}
}
