package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/infrastructure/util"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
)

type CommandRollback struct{}

func (c *CommandRollback) Name() string {
	return "rollback"
}

func (c *CommandRollback) Description() string {
	return "Удаляет последние <n> сообщений из истории разговоров для текущего чата. Если <n> не указано, то удаляется одно сообщение."
}

func (c *CommandRollback) IsAdmin() bool {
	return false
}

func (c *CommandRollback) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	session := chat.ActiveSession()
	number := 1
	if len(ctx.CommandArgs) > 0 {
		var err error
		number, err = strconv.Atoi(ctx.CommandArgs)
		if err != nil || number < 1 {
			number = 1
		}
	}

	if len(session.History) == 0 {
		return reply("История разговоров пуста.")
	}

	removed := service.RollbackHistory(session, number)
	return reply(fmt.Sprintf("Удалено %d %s.", removed, util.Pluralize(removed, [3]string{"сообщение", "сообщения", "сообщений"})))
}
