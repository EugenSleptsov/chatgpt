package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
	"strconv"
)

type CommandAdminRemoveUser struct {
	*Deps
}

func (c *CommandAdminRemoveUser) Name() string {
	return "removeuser"
}

func (c *CommandAdminRemoveUser) Description() string {
	return "Удаляет пользователя из авторизованных."
}

func (c *CommandAdminRemoveUser) IsAdmin() bool {
	return true
}

func (c *CommandAdminRemoveUser) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	if len(ctx.Msg.CommandArguments()) == 0 {
		return reply("Укажите ID пользователя. Использование: /removeuser <id>")
	}

	userId, err := strconv.ParseInt(ctx.Msg.CommandArguments(), 10, 64)
	if err != nil {
		return reply(fmt.Sprintf("Некорректный ID: %s", ctx.Msg.CommandArguments()))
	}

	var responses []handler.Response
	newList := make([]int64, 0)
	for _, id := range c.Auth.GetAuthorizedUsers() {
		if id == userId {
			responses = append(responses, handler.Response{Text: fmt.Sprintf("Пользователь будет удалён: %d", userId)})
		} else {
			newList = append(newList, id)
		}
	}

	c.Auth.SetAuthorizedUsers(newList)
	c.Config.AuthorizedUserIds = c.Auth.GetAuthorizedUsers()
	if err = conf.UpdateConfig(c.ConfigPath, c.Config); err != nil {
		c.Notifier.LogError(err)
		return append(responses, handler.Response{Text: fmt.Sprintf("Ошибка сохранения конфига: %v", err)})
	}

	responses = append(responses, handler.Response{Text: "Пользователь удалён."})
	return responses
}
