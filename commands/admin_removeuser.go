package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	conf "GPTBot/config"
	"GPTBot/storage"
	"fmt"
	"log"
	"strconv"
)

type CommandAdminRemoveUser struct{}

func (c *CommandAdminRemoveUser) Name() string {
	return "adduser"
}

func (c *CommandAdminRemoveUser) Description() string {
	return "Добавляет пользователя в авторизованные."
}

func (c *CommandAdminRemoveUser) IsAdmin() bool {
	return true
}

func (c *CommandAdminRemoveUser) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to remove")
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		newList := make([]int64, 0)
		for _, auth := range bot.Config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User will be removed: %d", userId))
			} else {
				newList = append(newList, auth)
			}
		}

		bot.Config.AuthorizedUserIds = newList
		err = conf.UpdateConfig("bot.conf", bot.Config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, "Command successfully ended")
	}
}
