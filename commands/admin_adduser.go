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

func (c *CommandAdminAddUser) Execute(update telegram.Update, chat *storage.Chat) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		c.Bot.Reply(chatID, update.Message.MessageID, "Укажите ID пользователя. Использование: /adduser <id>")
		return
	}

	userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
	if err != nil {
		c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Некорректный ID: %s", update.Message.CommandArguments()))
		return
	}

	for _, id := range c.Auth.GetAuthorizedUsers() {
		if id == userId {
			c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Пользователь уже добавлен: %d", userId))
			return
		}
	}

	newList := append(c.Auth.GetAuthorizedUsers(), userId)
	c.Auth.SetAuthorizedUsers(newList)
	// Update config snapshot under the same write so reload sees the change.
	c.Config.AuthorizedUserIds = c.Auth.GetAuthorizedUsers()
	if err = conf.UpdateConfig(c.ConfigPath, c.Config); err != nil {
		c.Notifier.LogError(err)
		c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Ошибка сохранения конфига: %v", err))
		return
	}

	c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Пользователь добавлен: %d", userId))
}
