package commands

import (
	"GPTBot/api/telegram"
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

func (c *CommandSessionRemove) Execute(update telegram.Update, chat *storage.Chat) {
	arg := strings.TrimSpace(update.Message.CommandArguments())
	if arg == "" {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Укажите ID сессии. Использование: /remove <id>")
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

	if !chat.RemoveSession(id) {
		c.Bot.Reply(chat.ChatID, update.Message.MessageID, "Нельзя удалить единственную сессию.")
		return
	}

	c.Bot.Reply(chat.ChatID, update.Message.MessageID, fmt.Sprintf("Сессия #%d (%s) удалена. Активная: #%d.", id, s.Topic, chat.ActiveSessionID))
}
