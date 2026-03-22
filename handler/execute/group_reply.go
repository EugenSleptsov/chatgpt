package execute

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/storage"
)

// GroupReplyExecutor handles IntentGroupReply — the bot was explicitly
// addressed in a group chat; replies from conversation history with tools.
type GroupReplyExecutor struct {
	Deps *commands.Deps
}

func (e *GroupReplyExecutor) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, _ *handler.Request) []handler.Response {
	e.Deps.Notifier.Logf("[Group] %s → бот упомянут, отвечаю", ctx.SenderName)

	result, err := e.Deps.GPTService.ReplyFromGroupHistory(chat)
	e.Deps.Notifier.LogError(err)
	e.Deps.Notifier.Logf("[GroupGPT] %s", result.Text)

	return chatResultToResponses(result, chat.Settings.UseMarkdown)
}
