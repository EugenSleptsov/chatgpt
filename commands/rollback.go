package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strconv"
)

type CommandRollback struct{}

func (c *CommandRollback) Name() string {
	return "rollback"
}

func (c *CommandRollback) Description() string {
	return "Rollback last message"
}

func (c *CommandRollback) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	number := 1
	if len(update.Message.CommandArguments()) > 0 {
		var err error
		number, err = strconv.Atoi(update.Message.CommandArguments())
		if err != nil || number < 1 {
			number = 1
		}
	}

	if number > len(chat.History) {
		number = len(chat.History)
	}

	if len(chat.History) > 0 {
		chat.History = chat.History[:len(chat.History)-number]
		bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Удалено %d %s.", number, util.Pluralize(number, [3]string{"сообщение", "сообщения", "сообщений"})))
	} else {
		bot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров пуста.")
	}
}
