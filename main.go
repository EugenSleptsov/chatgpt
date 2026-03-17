package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	conf "GPTBot/config"
	"GPTBot/handler"
	"GPTBot/manager"
	"GPTBot/storage"
)

const (
	numWorkers       = 10
	updateBufferSize = 100
)

func main() {
	logSystem := log.NewSystem()

	config, err := conf.ReadConfig("bot.yaml")
	logSystem.LogFatal(err)
	gptClient := gpt.NewGPTClient(config.GPTToken)

	telegramBot, err := telegram.NewInstance(config, logSystem)
	logSystem.LogFatal(err)

	commandFactory := commands.NewCommandFactory()
	deps := &commands.Deps{Bot: telegramBot, GptClient: gptClient, Registry: commandFactory}
	registerCommands(commandFactory, deps)

	botStorage, err := storage.NewFileStorage("data")
	logSystem.LogFatal(err)

	startWorkers(
		telegramBot,
		manager.NewTelegramChatManager(botStorage, telegramBot.Config, logSystem),
		commandFactory,
		handler.NewUpdateHandlerFactory(telegramBot, commandFactory, gptClient, logSystem, logSystem),
	)
}

func startWorkers(
	telegramBot *telegram.Bot,
	chatManager manager.ChatManager,
	commandFactory commands.CommandFactory,
	handlerFactory handler.UpdateHandlerFactory,
) {
	updateChan := make(chan telegram.Update, updateBufferSize)
	for i := 0; i < numWorkers; i++ {
		worker := NewWorker(telegramBot, chatManager, commandFactory, handlerFactory)
		go worker.Start(updateChan)
	}
	for update := range telegramBot.GetUpdateChannel(telegramBot.Config.TimeoutValue) {
		updateChan <- update
	}
}

func registerCommands(factory commands.CommandFactory, d *commands.Deps) {
	factory.Register("help", func() commands.Command { return &commands.CommandHelp{Deps: d} })
	factory.Register("start", func() commands.Command { return &commands.CommandStart{Deps: d} })
	factory.Register("clear", func() commands.Command { return &commands.CommandClear{Deps: d} })
	factory.Register("history", func() commands.Command { return &commands.CommandHistory{Deps: d} })
	factory.Register("rollback", func() commands.Command { return &commands.CommandRollback{Deps: d} })
	factory.Register("translate", func() commands.Command { return &commands.CommandTranslate{Deps: d} })
	factory.Register("tech_translate", func() commands.Command { return &commands.CommandTechTranslate{Deps: d} })
	factory.Register("enhance", func() commands.Command { return &commands.CommandEnhance{Deps: d} })
	factory.Register("grammar", func() commands.Command { return &commands.CommandGrammar{Deps: d} })
	factory.Register("summarize", func() commands.Command { return &commands.CommandSummarize{Deps: d} })
	factory.Register("summarize_prompt", func() commands.Command { return &commands.CommandSummarizePrompt{Deps: d} })
	factory.Register("analyze", func() commands.Command { return &commands.CommandAnalyze{Deps: d} })
	factory.Register("temperature", func() commands.Command { return &commands.CommandTemperature{Deps: d} })
	factory.Register("model", func() commands.Command { return &commands.CommandModel{Deps: d} })
	factory.Register("imagine", func() commands.Command { return &commands.CommandImagine{Deps: d} })
	factory.Register("system", func() commands.Command { return &commands.CommandSystem{Deps: d} })
	factory.Register("markdown", func() commands.Command { return &commands.CommandMarkdown{Deps: d} })
	factory.Register("reload", func() commands.Command { return &commands.CommandAdminReload{Deps: d} })
	factory.Register("adduser", func() commands.Command { return &commands.CommandAdminAddUser{Deps: d} })
	factory.Register("removeuser", func() commands.Command { return &commands.CommandAdminRemoveUser{Deps: d} })
}
