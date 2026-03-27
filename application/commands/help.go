package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandHelp struct {
	Registry *Registry
	Auth     *service.Auth
}

func (c *CommandHelp) Name() string {
	return "help"
}

func (c *CommandHelp) Description() string {
	return "Показывает список доступных команд и их описание."
}

func (c *CommandHelp) IsAdmin() bool {
	return false
}

func (c *CommandHelp) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	CommandList := c.Registry.All()

	message := "Список доступных команд и их описание:\n"
	var adminCommands []Command
	for _, command := range CommandList {
		if command.IsAdmin() {
			adminCommands = append(adminCommands, command)
			continue
		}

		message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
	}

	if c.Auth.IsAdmin(ctx.SenderID) {
		message += "\nКоманды администратора:\n"
		for _, command := range adminCommands {
			message += fmt.Sprintf("/%s - %s\n", command.Name(), command.Description())
		}
	}

	return reply(message)
}
