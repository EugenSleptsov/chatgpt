package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"log"
)

func main() {
	config, err := conf.ReadConfig("bot.conf")
	handleError(err, "Error reading config file")

	bot, err := telegram.NewInstance(config)
	handleError(err, "Error creating Telegram bot")

	botStorage, err := storage.NewFileStorage("data")
	handleError(err, "Error creating storage")

	gptClient := gpt.NewGPTClient(config.GPTToken)

	start(bot, gptClient, botStorage)
}

func handleError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
