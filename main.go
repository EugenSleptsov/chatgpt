package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"log"
)

func main() {
	config, err := readConfig("bot.conf")
	handleError(err, "Error reading config file")

	bot, err := telegram.NewBot(config.TelegramToken, config.CommandMenu)
	handleError(err, "Error creating Telegram bot")
	bot.SetAdminId(config.AdminId)

	botStorage, err := storage.NewFileStorage("data")
	handleError(err, "Error creating storage")

	gptClient := gpt.NewGPTClient(config.GPTToken)

	start(bot, gptClient, botStorage, config)
}

func handleError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
