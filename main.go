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

	config, err := conf.ReadConfig("bot.conf")
	logSystem.LogFatal(err)
	gptClient := gpt.NewGPTClient(config.GPTToken)

	telegramBot, err := telegram.NewInstance(config, logSystem)
	logSystem.LogFatal(err)

	commandFactory := commands.NewCommandFactory()
	registerCommands(commandFactory, telegramBot, commandFactory, gptClient)

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

func registerCommands(commandFactory commands.CommandFactory, telegramBot *telegram.Bot, commandRegistry commands.CommandRegistry, gptClient gpt.Client) {
	commandFactory.Register("help", func() commands.Command {
		return &commands.CommandHelp{TelegramBot: telegramBot, CommandRegistry: commandRegistry}
	})
	commandFactory.Register("start", func() commands.Command { return &commands.CommandStart{TelegramBot: telegramBot} })
	commandFactory.Register("clear", func() commands.Command { return &commands.CommandClear{TelegramBot: telegramBot} })
	commandFactory.Register("history", func() commands.Command { return &commands.CommandHistory{TelegramBot: telegramBot} })
	commandFactory.Register("rollback", func() commands.Command { return &commands.CommandRollback{TelegramBot: telegramBot} })
	commandFactory.Register("translate", func() commands.Command {
		return &commands.CommandTranslate{TelegramBot: telegramBot, GptClient: gptClient}
	})
	commandFactory.Register("enhance", func() commands.Command {
		return &commands.CommandEnhance{TelegramBot: telegramBot, GptClient: gptClient}
	})
	commandFactory.Register("grammar", func() commands.Command {
		return &commands.CommandGrammar{TelegramBot: telegramBot, GptClient: gptClient}
	})
	commandFactory.Register("summarize", func() commands.Command {
		return &commands.CommandSummarize{TelegramBot: telegramBot, GptClient: gptClient}
	})
	commandFactory.Register("summarize_prompt", func() commands.Command { return &commands.CommandSummarizePrompt{TelegramBot: telegramBot} })
	commandFactory.Register("analyze", func() commands.Command {
		return &commands.CommandAnalyze{TelegramBot: telegramBot, GptClient: gptClient}
	})
	commandFactory.Register("temperature", func() commands.Command { return &commands.CommandTemperature{TelegramBot: telegramBot} })
	commandFactory.Register("model", func() commands.Command { return &commands.CommandModel{TelegramBot: telegramBot} })
	commandFactory.Register("imagine", func() commands.Command {
		return &commands.CommandImagine{TelegramBot: telegramBot, GptClient: gptClient}
	})
	commandFactory.Register("system", func() commands.Command { return &commands.CommandSystem{TelegramBot: telegramBot} })
	commandFactory.Register("markdown", func() commands.Command { return &commands.CommandMarkdown{TelegramBot: telegramBot} })

	commandFactory.Register("reload", func() commands.Command { return &commands.CommandAdminReload{TelegramBot: telegramBot} })
	commandFactory.Register("adduser", func() commands.Command { return &commands.CommandAdminAddUser{TelegramBot: telegramBot} })
	commandFactory.Register("removeuser", func() commands.Command { return &commands.CommandAdminRemoveUser{TelegramBot: telegramBot} })
}
