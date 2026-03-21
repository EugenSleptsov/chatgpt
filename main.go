package main

import (
	"GPTBot/api/gpt/openai"
	"GPTBot/api/logger"
	"GPTBot/api/telegram"
	"GPTBot/api/telegram/adminlog"
	"GPTBot/commands"
	conf "GPTBot/config"
	"GPTBot/handler"
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
	router := handler.NewRouter(deps)

	startWorkers(deps, chatManager, router, telegramBot.GetUpdateChannel(config.TimeoutValue))
}

func startWorkers(
	deps *commands.Deps,
	chatManager manager.ChatManager,
	router *handler.Router,
	telegramUpdates telegram.UpdatesChannel,
) {
	updateChan := make(chan telegram.Update, updateBufferSize)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		worker := NewWorker(deps, chatManager, router)
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
