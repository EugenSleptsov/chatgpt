package execute

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/handler"
	"GPTBot/storage"
)

// GroupAutoReplyExecutor handles IntentGroupAutoReply — evaluates whether
// the bot should proactively join a group conversation (via GPT check).
type GroupAutoReplyExecutor struct {
	Deps *commands.Deps
}

func (e *GroupAutoReplyExecutor) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, _ *handler.Request) []handler.Response {
	if !e.Deps.Auth.IsAuthorized(ctx.SenderID) {
		return nil
	}

	should, reason, err := e.Deps.GPTService.ShouldAutoReply(chat)
	e.Deps.Notifier.LogError(err)

	if !should {
		e.Deps.Notifier.Logf("[Group] Авто-ответ: НЕТ (%s)", reason)
		return nil
	}

	e.Deps.Notifier.Logf("[Group] Авто-ответ: ДА (%s)", reason)
	result, err := e.Deps.GPTService.ReplyFromGroupHistory(chat)
	e.Deps.Notifier.LogError(err)
	e.Deps.Notifier.Logf("[GroupGPT] %s", result.Text)
	return chatResultToResponses(result, chat.Settings.UseMarkdown)
}
