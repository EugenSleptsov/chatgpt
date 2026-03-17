package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"fmt"
	"log"
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
		c.Bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to add")
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		for _, auth := range c.Bot.Config.AuthorizedUserIds {
			if auth == userId {
				c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User already added: %d", userId))
				return
			}
		}

		c.Bot.Config.AuthorizedUserIds = append(c.Bot.Config.AuthorizedUserIds, userId)
		err = conf.UpdateConfig("bot.yaml", c.Bot.Config)
		if err != nil {
			log.Fatalf("Error updating bot.yaml: %v", err)
		}

		c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User successfully added: %d", userId))
	}
}
