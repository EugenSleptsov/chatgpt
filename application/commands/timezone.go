package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
	"time"
)

// CommandTimezone sets the chat's timezone (IANA name) used for reminders and
// all time display. Typed use: /timezone Europe/Berlin. Also the target of the
// settings-hub ForceReply flow (PendingInput = "timezone").
type CommandTimezone struct{}

func (c *CommandTimezone) Name() string { return "timezone" }
func (c *CommandTimezone) Description() string {
	return "Часовой пояс чата для напоминаний. Использование: /timezone Europe/Berlin"
}
func (c *CommandTimezone) IsAdmin() bool { return false }

func (c *CommandTimezone) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	arg := strings.TrimSpace(ctx.CommandArgs)
	if arg == "" {
		return reply(fmt.Sprintf("Часовой пояс: %s\nСейчас: %s",
			timezoneLabel(ch), time.Now().In(ch.Location()).Format("02.01 15:04")))
	}
	if _, err := time.LoadLocation(arg); err != nil {
		return reply(fmt.Sprintf("Не знаю зону «%s». Нужно имя IANA, например Europe/Berlin, Europe/Moscow, Asia/Tbilisi.", arg))
	}
	ch.Settings.Timezone = arg
	return reply(fmt.Sprintf("Часовой пояс установлен: %s (сейчас %s)",
		arg, time.Now().In(ch.Location()).Format("02.01 15:04")))
}

// timezoneLabel renders the chat's timezone for UI ("серверный" when unset).
func timezoneLabel(ch *chat.Chat) string {
	if ch.Settings.Timezone == "" {
		return "серверный (" + time.Now().Format("MST") + ")"
	}
	return ch.Settings.Timezone
}
