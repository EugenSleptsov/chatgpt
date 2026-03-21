package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strconv"
)

type CommandRollback struct {
	*Deps
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

func (c *CommandRollback) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) {
	session := chat.ActiveSession()
	number := 1
	if len(ctx.Msg.CommandArguments()) > 0 {
		var err error
		number, err = strconv.Atoi(ctx.Msg.CommandArguments())
		if err != nil || number < 1 {
			number = 1
		}
	}

	if number > len(session.History) {
		number = len(session.History)
	}

	if len(session.History) > 0 {
		session.History = session.History[:len(session.History)-number]
		c.Bot.Reply(chat.ChatID, ctx.MessageID, fmt.Sprintf("Удалено %d %s.", number, util.Pluralize(number, [3]string{"сообщение", "сообщения", "сообщений"})))
	} else {
		c.Bot.Reply(chat.ChatID, ctx.MessageID, "История разговоров пуста.")
	}
}
