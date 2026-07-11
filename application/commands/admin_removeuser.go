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

	users := c.Auth.GetAuthorizedUsers()
	newList := make([]int64, 0, len(users))
	found := false
	for _, id := range users {
		if id == userId {
			found = true
		} else {
			newList = append(newList, id)
		}
	}

	if !found {
		return reply(fmt.Sprintf("Пользователь %d не найден в списке.", userId))
	}
	// Пустой список означает публичный доступ (см. Auth.IsAuthorized), поэтому
	// последнего пользователя удалить нельзя — иначе бот откроется всем.
	if len(newList) == 0 {
		return reply("Нельзя удалить последнего авторизованного пользователя: пустой список открыл бы бота всем.")
	}

	responses := []sender.Response{{Text: fmt.Sprintf("Пользователь будет удалён: %d", userId)}}
	c.Auth.SetAuthorizedUsers(newList)
	if err = c.ConfigService.SetAuthorizedUsers(c.Auth.GetAuthorizedUsers()); err != nil {
		c.Notifier.LogError(err)
		return append(responses, sender.Response{Text: fmt.Sprintf("Ошибка сохранения конфига: %v", err)})
	}

	responses = append(responses, sender.Response{Text: "Пользователь удалён."})
	return responses
}
