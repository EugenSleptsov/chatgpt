package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

type CommandHelp struct {
	Registry *Registry
	Auth     *service.Auth
}

func (c *CommandHelp) Name() string {
	return "help"
}

func (c *CommandHelp) Description() string {
	return "Главное меню (кнопки). /help list — текстовый список команд."
}

func (c *CommandHelp) IsAdmin() bool {
	return false
}

// Execute shows the button launcher by default. The full text command list is
// available behind "help:list" (a button in the menu's Info section) or by
// typing "/help list".
func (c *CommandHelp) Execute(ctx *pipeline.RequestContext, _ *chat.Chat) []sender.Response {
	if strings.TrimSpace(ctx.CommandArgs) != "list" {
		return mainMenuView()
	}

	message := "Список доступных команд и их описание:\n"
	var adminCommands []Command
	for _, command := range c.Registry.All() {
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

	return []sender.Response{{
		Text:    message,
		Buttons: [][]sender.Button{{{Text: "⬅ Назад", Data: "menu:info"}}},
	}}
}
