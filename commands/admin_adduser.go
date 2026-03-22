package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
	"strconv"
)

type CommandAdminAddUser struct {
	*Deps
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

func (c *CommandAdminAddUser) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	if len(ctx.Msg.CommandArguments()) == 0 {
		return reply("Укажите ID пользователя. Использование: /adduser <id>")
	}

	userId, err := strconv.ParseInt(ctx.Msg.CommandArguments(), 10, 64)
	if err != nil {
		return reply(fmt.Sprintf("Некорректный ID: %s", ctx.Msg.CommandArguments()))
	}

	for _, id := range c.Auth.GetAuthorizedUsers() {
		if id == userId {
			return reply(fmt.Sprintf("Пользователь уже добавлен: %d", userId))
		}
	}

	newList := append(c.Auth.GetAuthorizedUsers(), userId)
	c.Auth.SetAuthorizedUsers(newList)
	c.Config.AuthorizedUserIds = c.Auth.GetAuthorizedUsers()
	if err = conf.UpdateConfig(c.ConfigPath, c.Config); err != nil {
		c.Notifier.LogError(err)
		return reply(fmt.Sprintf("Ошибка сохранения конфига: %v", err))
	}

	return reply(fmt.Sprintf("Пользователь добавлен: %d", userId))
}
