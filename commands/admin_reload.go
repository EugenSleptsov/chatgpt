package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"fmt"
	"log"
)

type CommandAdminReload struct {
	*Deps
}

func (c *CommandAdminReload) Name() string {
	return "reload"
}

func (c *CommandAdminReload) Description() string {
	return "Перезагружает конфигурационный файл."
}

func (c *CommandAdminReload) IsAdmin() bool {
	return true
}

func (c *CommandAdminReload) Execute(update telegram.Update, chat *storage.Chat) {
	chatID := chat.ChatID

	config, err := conf.ReadConfig("bot.yaml")
	if err != nil {
		log.Fatalf("Error reading bot.yaml: %v", err)
	}

	c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(config)))
}
