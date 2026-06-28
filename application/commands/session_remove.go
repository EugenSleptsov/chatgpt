package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
	"strings"
)

type CommandSessionRemove struct{}

func (c *CommandSessionRemove) Name() string {
	return "remove"
}

func (c *CommandSessionRemove) Description() string {
	return "Удаляет сессию по ID. Нельзя удалить последнюю. Использование: /remove <id>"
}

func (c *CommandSessionRemove) IsAdmin() bool {
	return false
}

// Execute supports both typed use and the button flow:
//
//	"<id>"      → show a delete confirmation for session <id>
//	"yes:<id>"  → perform the delete, then re-render the session list
//	"cancel"    → re-render the session list (abort)
func (c *CommandSessionRemove) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	arg := strings.TrimSpace(ctx.CommandArgs)

	switch {
	case arg == "cancel":
		return sessionListView(chat, sessionPageOf(chat, chat.ActiveSessionID))
	case strings.HasPrefix(arg, "yes:"):
		id, err := strconv.Atoi(arg[len("yes:"):])
		if err != nil {
			return reply("ID должен быть числом.")
		}
		return c.performRemove(chat, id)
	}

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

	if chat.Settings.SkipDeleteConfirm {
		return c.performRemove(chat, id)
	}

	return []sender.Response{{
		Text: fmt.Sprintf("Удалить сессию #%d (%s)?", id, s.Topic),
		Buttons: [][]sender.Button{{
			{Text: fmt.Sprintf("🗑 Удалить #%d", id), Data: fmt.Sprintf("remove:yes:%d", id)},
			{Text: "Отмена", Data: "remove:cancel"},
		}},
	}}
}

// performRemove deletes the session and returns the refreshed list view (so the
// user stays inside the session hub).
func (c *CommandSessionRemove) performRemove(chat *chat.Chat, id int) []sender.Response {
	s := chat.FindSession(id)
	if s == nil {
		return reply(fmt.Sprintf("Сессия #%d не найдена.", id))
	}
	if !chat.RemoveSession(id) {
		return reply("Нельзя удалить единственную сессию.")
	}
	return sessionListView(chat, sessionPageOf(chat, chat.ActiveSessionID))
}
