package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"fmt"
	"strconv"
	"strings"
)

type CommandSessionRemove struct {
	*Deps
}

func (c *CommandSessionRemove) Name() string {
	return "remove"
}

func (c *CommandSessionRemove) Description() string {
	return "Удаляет сессию по ID. Нельзя удалить последнюю. Использование: /remove <id>"
}

func (c *CommandSessionRemove) IsAdmin() bool {
	return false
}

func (c *CommandSessionRemove) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	arg := strings.TrimSpace(ctx.Msg.CommandArguments())
	if arg == "" {
		return reply("Укажите ID сессии. Использование: /remove <id>")
	}

	id, err := strconv.Atoi(arg)
	if err != nil {
		return reply("ID должен быть числом.")
	}

	s := chat.FindSession(id)
	if s == nil {
		return reply(fmt.Sprintf("Сессия #%d не найдена.", id))
	}

	if !chat.RemoveSession(id) {
		return reply("Нельзя удалить единственную сессию.")
	}

	return reply(fmt.Sprintf("Сессия #%d (%s) удалена. Активная: #%d.", id, s.Topic, chat.ActiveSessionID))
}
