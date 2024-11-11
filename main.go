package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	conf "GPTBot/config"
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

	bot, err := telegram.NewInstance(config)
	logClient.LogFatal(err)

	botStorage, err := storage.NewFileStorage("data")
	logClient.LogFatal(err)

	gptClient := gpt.NewGPTClient(config.GPTToken)

	Init(bot, gptClient, botStorage, logClient)
}

func Init(bot *telegram.Bot, gptClient *gpt.GPTClient, botStorage storage.Storage, logClient *log.Log) {
	updateChan := make(chan telegram.Update, updateBufferSize)
	for i := 0; i < numWorkers; i++ {
		worker := NewWorker(bot, gptClient, botStorage, logClient)
		go worker.Start(updateChan)
	}

	for update := range bot.GetUpdateChannel(bot.Config.TimeoutValue) {
		updateChan <- update
	}
}
