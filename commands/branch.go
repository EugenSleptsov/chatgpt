package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/service"
	"GPTBot/storage"
)

// Branch implements handler.CommandBranch: it looks up the command by
// name, checks admin permissions, and delegates to Execute.
type Branch struct {
	Registry *Registry
	Auth     *service.Auth
	Notifier *service.Notifier
}

// compile-time check
var _ handler.CommandBranch = (*Branch)(nil)

func (b *Branch) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *handler.Request) []handler.Response {
	cmd, err := b.Registry.Get(req.CommandName)
	if err != nil {
		b.Notifier.Logf("Command not found: %v", err)
		return nil
	}
	if cmd.IsAdmin() && !b.Auth.IsAdmin(ctx.SenderID) {
		return nil
	}
	return cmd.Execute(ctx, chat)
}
