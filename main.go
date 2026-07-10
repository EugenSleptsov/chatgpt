package main

import (
	"GPTBot/app"
	"log"
	// Embed the IANA tz database so chat timezones resolve on hosts without
	// system tzdata (Windows, scratch containers).
	_ "time/tzdata"
)

const configFile = "config/bot.yaml"

func main() {
	core, err := app.NewApp(configFile)
	if err != nil {
		log.Fatal(err)
	}
	core.Run()
}
