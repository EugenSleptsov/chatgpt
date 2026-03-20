package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
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

func (c *CommandAdminRemoveUser) Execute(update telegram.Update, chat *storage.Chat) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		c.Bot.Reply(chatID, update.Message.MessageID, "Укажите ID пользователя. Использование: /removeuser <id>")
		return
	}

	userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
	if err != nil {
		c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Некорректный ID: %s", update.Message.CommandArguments()))
		return
	}

	newList := make([]int64, 0)
	for _, id := range c.Auth.GetAuthorizedUsers() {
		if id == userId {
			c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Пользователь будет удалён: %d", userId))
		} else {
			newList = append(newList, id)
		}
	}

	c.Auth.SetAuthorizedUsers(newList)
	c.Config.AuthorizedUserIds = c.Auth.GetAuthorizedUsers()
	if err = conf.UpdateConfig(c.ConfigPath, c.Config); err != nil {
		c.Notifier.LogError(err)
		c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Ошибка сохранения конфига: %v", err))
		return
	}

	c.Bot.Reply(chatID, update.Message.MessageID, "Пользователь удалён.")
}
