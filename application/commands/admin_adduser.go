package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
)

type CommandAdminAddUser struct {
	ConfigService *service.ConfigService
	Auth          *service.Auth
	Notifier      *service.Notifier
}

func (c *CommandAdminAddUser) Name() string {
	return "adduser"
}

func (c *CommandAdminAddUser) Description() string {
	return "Добавляет пользователя в авторизованные."
}

func (c *CommandAdminAddUser) IsAdmin() bool {
	return true
}

func (c *CommandAdminAddUser) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if len(ctx.CommandArgs) == 0 {
		return reply("Укажите ID пользователя. Использование: /adduser <id>")
	}

	userId, err := strconv.ParseInt(ctx.CommandArgs, 10, 64)
	if err != nil {
		return reply(fmt.Sprintf("Некорректный ID: %s", ctx.CommandArgs))
	}

	for _, id := range c.Auth.GetAuthorizedUsers() {
		if id == userId {
			return reply(fmt.Sprintf("Пользователь уже добавлен: %d", userId))
		}
	}

	newList := append(c.Auth.GetAuthorizedUsers(), userId)
	c.Auth.SetAuthorizedUsers(newList)
	if err = c.ConfigService.SetAuthorizedUsers(c.Auth.GetAuthorizedUsers()); err != nil {
		c.Notifier.LogError(err)
		return reply(fmt.Sprintf("Ошибка сохранения конфига: %v", err))
	}

	return reply(fmt.Sprintf("Пользователь добавлен: %d", userId))
}
