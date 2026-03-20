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
)

const (
	numWorkers       = 10
	updateBufferSize = 100
)

func main() {
	logSystem := logger.NewSystem()

	config, err := conf.ReadConfig("bot.yaml")
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

	auth := &service.Auth{
		AdminID:           config.AdminId,
		AuthorizedUserIDs: config.AuthorizedUserIds,
	}

	gptService := &service.GPTService{
		GptClient: openai.NewClient(config.GPTToken, logSystem),
	}

	deps := &commands.Deps{
		Bot:        telegramBot,
		Config:     config,
		Registry:   commands.NewCommandFactory(),
		GPTService: gptService,
		Notifier:   notifier,
		Auth:       auth,
	}
	registerCommands(deps)

	botStorage, err := storage.NewFileStorage("data")
	logSystem.LogFatal(err)

	chatManager := manager.NewTelegramChatManager(botStorage, config, logSystem)
	handlerFactory := handler.NewUpdateHandlerFactory(deps)

	startWorkers(deps, chatManager, handlerFactory)
}

func startWorkers(
	deps *commands.Deps,
	chatManager manager.ChatManager,
	handlerFactory handler.UpdateHandlerFactory,
) {
	updateChan := make(chan telegram.Update, updateBufferSize)
	for i := 0; i < numWorkers; i++ {
		worker := NewWorker(deps, chatManager, handlerFactory)
		go worker.Start(updateChan)
	}
	for update := range deps.Bot.GetUpdateChannel(deps.Config.TimeoutValue) {
		updateChan <- update
	}
}

func registerCommands(d *commands.Deps) {
	d.Registry.Register("help", func() commands.Command { return &commands.CommandHelp{Deps: d} })
	d.Registry.Register("start", func() commands.Command { return &commands.CommandStart{Deps: d} })
	d.Registry.Register("clear", func() commands.Command { return &commands.CommandClear{Deps: d} })
	d.Registry.Register("history", func() commands.Command { return &commands.CommandHistory{Deps: d} })
	d.Registry.Register("rollback", func() commands.Command { return &commands.CommandRollback{Deps: d} })
	d.Registry.Register("translate", func() commands.Command { return &commands.CommandTranslate{Deps: d} })
	d.Registry.Register("tech_translate", func() commands.Command { return &commands.CommandTechTranslate{Deps: d} })
	d.Registry.Register("enhance", func() commands.Command { return &commands.CommandEnhance{Deps: d} })
	d.Registry.Register("grammar", func() commands.Command { return &commands.CommandGrammar{Deps: d} })
	d.Registry.Register("summarize", func() commands.Command { return &commands.CommandSummarize{Deps: d} })
	d.Registry.Register("summarize_prompt", func() commands.Command { return &commands.CommandSummarizePrompt{Deps: d} })
	d.Registry.Register("analyze", func() commands.Command { return &commands.CommandAnalyze{Deps: d} })
	d.Registry.Register("temperature", func() commands.Command { return &commands.CommandTemperature{Deps: d} })
	d.Registry.Register("model", func() commands.Command { return &commands.CommandModel{Deps: d} })
	d.Registry.Register("imagine", func() commands.Command { return &commands.CommandImagine{Deps: d} })
	d.Registry.Register("system", func() commands.Command { return &commands.CommandSystem{Deps: d} })
	d.Registry.Register("markdown", func() commands.Command { return &commands.CommandMarkdown{Deps: d} })
	d.Registry.Register("reload", func() commands.Command { return &commands.CommandAdminReload{Deps: d} })
	d.Registry.Register("adduser", func() commands.Command { return &commands.CommandAdminAddUser{Deps: d} })
	d.Registry.Register("removeuser", func() commands.Command { return &commands.CommandAdminRemoveUser{Deps: d} })
	d.Registry.Register("list", func() commands.Command { return &commands.CommandSessionList{Deps: d} })
	d.Registry.Register("current", func() commands.Command { return &commands.CommandSessionCurrent{Deps: d} })
	d.Registry.Register("use", func() commands.Command { return &commands.CommandSessionUse{Deps: d} })
	d.Registry.Register("new", func() commands.Command { return &commands.CommandSessionNew{Deps: d} })
	d.Registry.Register("remove", func() commands.Command { return &commands.CommandSessionRemove{Deps: d} })
	d.Registry.Register("update", func() commands.Command { return &commands.CommandSessionUpdate{Deps: d} })
}
