package executor

import (
	"GPTBot/application/commands"
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

// CommandExecutor matches bot commands and delegates to the command registry.
type CommandExecutor struct {
	Registry *commands.Registry
	Auth     *service.Auth
	Notifier *service.Notifier
}

func (e *CommandExecutor) Match(ctx *pipeline.RequestContext) bool {
	return ctx.IsCommand
}

func (e *CommandExecutor) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	cmd, err := e.Registry.Get(ctx.CommandName)
	if err != nil {
		e.Notifier.Logf("Command not found: %v", err)
		return nil
	}
	if cmd.IsAdmin() && !e.Auth.IsAdmin(ctx.SenderID) {
		return nil
	}
	return cmd.Execute(ctx, chat)
}
