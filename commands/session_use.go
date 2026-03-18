package commands

import (
	"GPTBot/api/telegram"
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

func (c *CommandSessionUse) Execute(update telegram.Update, chat *storage.Chat) {
	arg := strings.TrimSpace(update.Message.CommandArguments())
	if arg == "" {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Укажите ID сессии. Использование: /use <id>")
		return
	}

	id, err := strconv.Atoi(arg)
	if err != nil {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "ID должен быть числом.")
		return
	}

	s := chat.FindSession(id)
	if s == nil {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Сессия #%d не найдена.", id))
		return
	}

	chat.ActiveSessionID = s.ID
	c.Bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Переключено на сессию #%d — %s.", s.ID, s.Topic))
}
