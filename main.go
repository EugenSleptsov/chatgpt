package main

import (
	"GPTBot/app"
	"log"
)

const configFile = "config/bot.yaml"

func main() {
	core, err := app.NewApp(configFile)
	if err != nil {
		log.Fatal(err)
	}
	core.Run()
}
