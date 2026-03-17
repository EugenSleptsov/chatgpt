package commands

import (
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"fmt"
	"log"
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
		c.Bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to remove")
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		newList := make([]int64, 0)
		for _, auth := range c.Bot.Config.AuthorizedUserIds {
			if auth == userId {
				c.Bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User will be removed: %d", userId))
			} else {
				newList = append(newList, auth)
			}
		}

		c.Bot.Config.AuthorizedUserIds = newList
		err = conf.UpdateConfig("bot.yaml", c.Bot.Config)
		if err != nil {
			log.Fatalf("Error updating bot.yaml: %v", err)
		}

		c.Bot.Reply(chatID, update.Message.MessageID, "Command successfully ended")
	}
}
