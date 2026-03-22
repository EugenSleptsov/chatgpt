package main

import (
	conf "GPTBot/config"
	"GPTBot/infrastructure/storage"
	"log"
)

const configFile = "bot.yaml"

func main() {
	cfg, err := conf.ReadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	dsn := cfg.StorageDSN
	if dsn == "" {
		dsn = cfg.DataDir + "/chats.db"
	}
	if err := storage.MigrateFileToSQLite(cfg.DataDir, dsn); err != nil {
		log.Fatal(err)
	}
}
