package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strconv"
)

type CommandRollback struct {
	TelegramBot *telegram.Bot
}

func (c *CommandRollback) Name() string {
	return "rollback"
}

func (c *CommandRollback) Description() string {
	return "Удаляет последние <n> сообщений из истории разговоров для текущего чата. Если <n> не указано, то удаляется одно сообщение."
}

func (c *CommandRollback) IsAdmin() bool {
	return false
}

func (c *CommandRollback) Execute(update telegram.Update, chat *storage.Chat) {
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
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Удалено %d %s.", number, util.Pluralize(number, [3]string{"сообщение", "сообщения", "сообщений"})))
	} else {
		c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "История разговоров пуста.")
	}
}
