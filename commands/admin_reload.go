package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"fmt"
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

	newConfig, err := conf.ReadConfig(c.ConfigPath)
	if err != nil {
		c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Ошибка чтения конфига: %v", err))
		return
	}

	*c.Config = *newConfig
	c.Auth.SetAuthorizedUsers(c.Config.AuthorizedUserIds)

	c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Config updated: %s", fmt.Sprint(c.Config)))
}
