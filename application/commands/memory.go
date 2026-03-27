package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

type CommandMemory struct {
	Memory *service.MemoryService
}

func (c *CommandMemory) Name() string {
	return "memory"
}

func (c *CommandMemory) Description() string {
	return "Показывает память бота о чате. /memory clear — очистить."
}

func (c *CommandMemory) IsAdmin() bool {
	return false
}

func (c *CommandMemory) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	args := strings.TrimSpace(ctx.CommandArgs)

	if args == "clear" {
		count := c.Memory.Clear(chat)
		return reply(fmt.Sprintf("Память очищена (%d фактов удалено).", count))
	}

	return reply(c.Memory.FormatDisplay(chat))
}
