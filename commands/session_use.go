package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
	"strconv"
	"strings"
)

type CommandSessionUse struct {
	*Deps
}

func (c *CommandSessionUse) Name() string {
	return "use"
}

func (c *CommandSessionUse) Description() string {
	return "Переключает на сессию с указанным ID. Использование: /use <id>"
}

func (c *CommandSessionUse) IsAdmin() bool {
	return false
}

func (c *CommandSessionUse) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	arg := strings.TrimSpace(ctx.Msg.CommandArguments())
	if arg == "" {
		return reply("Укажите ID сессии. Использование: /use <id>")
	}

	id, err := strconv.Atoi(arg)
	if err != nil {
		return reply("ID должен быть числом.")
	}

	s := chat.FindSession(id)
	if s == nil {
		return reply(fmt.Sprintf("Сессия #%d не найдена.", id))
	}

	chat.ActiveSessionID = s.ID
	return reply(fmt.Sprintf("Переключено на сессию #%d — %s.", s.ID, s.Topic))
}
