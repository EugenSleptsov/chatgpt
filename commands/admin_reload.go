package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/handler"
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

func (c *CommandAdminReload) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	newConfig, err := conf.ReadConfig(c.ConfigPath)
	if err != nil {
		return reply(fmt.Sprintf("Ошибка чтения конфига: %v", err))
	}

	*c.Config = *newConfig
	c.Auth.SetAuthorizedUsers(c.Config.AuthorizedUserIds)

	return reply(fmt.Sprintf("Config updated: %s", fmt.Sprint(c.Config)))
}
