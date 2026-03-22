package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
)

type CommandAdminRemoveUser struct {
	ConfigService *service.ConfigService
	Auth          *service.Auth
	Notifier      *service.Notifier
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

func (c *CommandAdminRemoveUser) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	if len(ctx.CommandArgs) == 0 {
		return reply("Укажите ID пользователя. Использование: /removeuser <id>")
	}

	userId, err := strconv.ParseInt(ctx.CommandArgs, 10, 64)
	if err != nil {
		return reply(fmt.Sprintf("Некорректный ID: %s", ctx.CommandArgs))
	}

	var responses []sender.Response
	newList := make([]int64, 0)
	for _, id := range c.Auth.GetAuthorizedUsers() {
		if id == userId {
			responses = append(responses, sender.Response{Text: fmt.Sprintf("Пользователь будет удалён: %d", userId)})
		} else {
			newList = append(newList, id)
		}
	}

	c.Auth.SetAuthorizedUsers(newList)
	if err = c.ConfigService.SetAuthorizedUsers(c.Auth.GetAuthorizedUsers()); err != nil {
		c.Notifier.LogError(err)
		return append(responses, sender.Response{Text: fmt.Sprintf("Ошибка сохранения конфига: %v", err)})
	}

	responses = append(responses, sender.Response{Text: "Пользователь удалён."})
	return responses
}
