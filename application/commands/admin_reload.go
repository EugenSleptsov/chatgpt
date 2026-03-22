package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandAdminReload struct {
	ConfigService *service.ConfigService
	Auth          *service.Auth
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

func (c *CommandAdminReload) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	newConfig, err := c.ConfigService.Reload()
	if err != nil {
		return reply(fmt.Sprintf("Ошибка чтения конфига: %v", err))
	}

	c.Auth.SetAuthorizedUsers(newConfig.AuthorizedUserIds)

	return reply(fmt.Sprintf("Config updated: %s", c.ConfigService.String()))
}
