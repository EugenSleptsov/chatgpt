package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

type CommandMarkdown struct{}

func (c *CommandMarkdown) Name() string {
	return "markdown"
}

func (c *CommandMarkdown) Description() string {
	return "Использование Markdown в сообщениях. Использование: /markdown <on|off>"
}

func (c *CommandMarkdown) IsAdmin() bool {
	return false
}

func (c *CommandMarkdown) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	switch ctx.CommandArgs {
	case "on":
		chat.Settings.UseMarkdown = true
	case "off":
		chat.Settings.UseMarkdown = false
	}
	return boolView("markdown", "Markdown", chat.Settings.UseMarkdown)
}
