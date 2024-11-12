package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	"GPTBot/commands"
	conf "GPTBot/config"
	"GPTBot/handler"
	"GPTBot/storage"
)

const (
	numWorkers       = 10
	updateBufferSize = 100
)

func main() {
	logClient := log.NewLog()

	config, err := conf.ReadConfig("bot.conf")
	logClient.LogFatal(err)

	telegramBot, err := telegram.NewInstance(config)
	logClient.LogFatal(err)

	botStorage, err := storage.NewFileStorage("data")
	logClient.LogFatal(err)

	gptClient := gpt.NewGPTClient(config.GPTToken)

	commandFactory := commands.NewCommandFactory()

	commandFactory.Register("help", func() commands.Command {
		return &commands.CommandHelp{TelegramBot: telegramBot, CommandRegistry: commandFactory}
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

	handlerFactory := handler.NewUpdateHandlerFactory(telegramBot, commandFactory, gptClient, logClient)

	updateChan := make(chan telegram.Update, updateBufferSize)
	for i := 0; i < numWorkers; i++ {
		worker := NewWorker(telegramBot, gptClient, botStorage, logClient, commandFactory, handlerFactory)
		go worker.Start(updateChan)
	}

	for update := range telegramBot.GetUpdateChannel(telegramBot.Config.TimeoutValue) {
		updateChan <- update
	}
}
