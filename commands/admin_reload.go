package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"fmt"
	"log"
)

type CommandAdminReload struct{}

func (c *CommandAdminReload) Name() string {
	return "reload"
}

func (c *CommandAdminReload) Description() string {
	return "Перезагружает конфигурационный файл."
}

func (c *CommandAdminReload) IsAdmin() bool {
	return true
}

func (c *CommandAdminReload) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	chatID := chat.ChatID

	config, err := conf.ReadConfig("bot.conf")
	if err != nil {
		log.Fatalf("Error reading bot.conf: %v", err)
	}

	bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)))
}
