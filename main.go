package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"log"
)

var botStorage storage.Storage

func main() {
	config, err := readConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot, err := telegram.NewBot(config.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}
	bot.SetCommandList(config.CommandMenu)

	gptClient := &gpt.GPTClient{
		ApiKey: config.GPTToken,
	}

	botStorage, err = storage.NewFileStorage("data")
	if err != nil {
		log.Fatalf("Error creating storage: %v", err)
	}

	start(bot, gptClient, botStorage, config)
}
