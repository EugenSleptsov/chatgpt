package main

import "log"

const configFile = "bot.yaml"

func main() {
	app, err := NewApp(configFile)
	if err != nil {
		log.Fatal(err)
	}
	app.Run()
}
