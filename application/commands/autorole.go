package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

// CommandAutoRole sets or displays the auto-reply persona for the current chat.
// When called without arguments it shows the active persona;
// with text it overrides it for this chat only.
type CommandAutoRole struct{}

func (c *CommandAutoRole) Name() string {
	return "autorole"
}

func (c *CommandAutoRole) Description() string {
	return "Устанавливает или показывает роль (персону) бота для авто-ответа. Использование: /autorole [текст роли]"
}

func (c *CommandAutoRole) IsAdmin() bool {
	return true
}

func (c *CommandAutoRole) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	if len(ctx.CommandArgs) == 0 {
		persona := ch.Settings.AutoReplyPersona
		if persona == "" {
			persona = service.DefaultAutoReplyPersona
		}
		return reply(fmt.Sprintf("Текущая роль авто-ответа:\n\n%s", persona))
	}

	if ctx.CommandArgs == "reset" {
		ch.Settings.AutoReplyPersona = ""
		return reply("🔄 Роль авто-ответа сброшена на значение по умолчанию.")
	}

	ch.Settings.AutoReplyPersona = ctx.CommandArgs
	if len(ch.Settings.AutoReplyPersona) > 2048 {
		ch.Settings.AutoReplyPersona = ch.Settings.AutoReplyPersona[:2048]
	}
	return reply(fmt.Sprintf("✅ Роль авто-ответа установлена:\n\n%s", ch.Settings.AutoReplyPersona))
}
