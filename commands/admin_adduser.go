package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
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

func (c *CommandAdminAddUser) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	chatID := chat.ChatID
	if len(ctx.Msg.CommandArguments()) == 0 {
		c.Bot.Reply(chatID, ctx.MessageID, "Укажите ID пользователя. Использование: /adduser <id>")
		return
	}

	userId, err := strconv.ParseInt(ctx.Msg.CommandArguments(), 10, 64)
	if err != nil {
		c.Bot.Reply(chatID, ctx.MessageID, fmt.Sprintf("Некорректный ID: %s", ctx.Msg.CommandArguments()))
		return
	}

	for _, id := range c.Auth.GetAuthorizedUsers() {
		if id == userId {
			c.Bot.Reply(chatID, ctx.MessageID, fmt.Sprintf("Пользователь уже добавлен: %d", userId))
			return
		}
	}

	newList := append(c.Auth.GetAuthorizedUsers(), userId)
	c.Auth.SetAuthorizedUsers(newList)
	// Update config snapshot under the same write so reload sees the change.
	c.Config.AuthorizedUserIds = c.Auth.GetAuthorizedUsers()
	if err = conf.UpdateConfig(c.ConfigPath, c.Config); err != nil {
		c.Notifier.LogError(err)
		c.Bot.Reply(chatID, ctx.MessageID, fmt.Sprintf("Ошибка сохранения конфига: %v", err))
		return
	}

	c.Bot.Reply(chatID, ctx.MessageID, fmt.Sprintf("Пользователь добавлен: %d", userId))
}
