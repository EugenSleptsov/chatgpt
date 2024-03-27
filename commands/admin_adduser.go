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

type CommandAdminAddUser struct{}

func (c *CommandAdminAddUser) Name() string {
	return "adduser"
}

func (c *CommandAdminAddUser) Description() string {
	return "Добавляет пользователя в авторизованные."
}

func (c *CommandAdminAddUser) IsAdmin() bool {
	return true
}

func (c *CommandAdminAddUser) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	chatID := chat.ChatID
	if len(update.Message.CommandArguments()) == 0 {
		bot.Reply(chatID, update.Message.MessageID, "Please provide a user id to add")
	} else {
		userId, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
		if err != nil {
			bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("Invalid user id: %s", update.Message.CommandArguments()))
			return
		}

		for _, auth := range bot.Config.AuthorizedUserIds {
			if auth == userId {
				bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User already added: %d", userId))
				return
			}
		}

		bot.Config.AuthorizedUserIds = append(bot.Config.AuthorizedUserIds, userId)
		err = conf.UpdateConfig("bot.conf", bot.Config)
		if err != nil {
			log.Fatalf("Error updating bot.conf: %v", err)
		}

		bot.Reply(chatID, update.Message.MessageID, fmt.Sprintf("User successfully added: %d", userId))
	}
}
