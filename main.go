package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/log"
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
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
	start(bot, gptClient, botStorage, logClient)
}
