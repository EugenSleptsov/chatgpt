package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/commands"
	"GPTBot/storage"
)

// GroupReplyExecutor handles IntentGroupReply — the bot was explicitly
// addressed in a group chat and should reply from conversation history.
// GPT may also call function tools (generate_image, generate_voice).
type GroupReplyExecutor struct {
	Deps *commands.Deps
}

func (e *GroupReplyExecutor) Execute(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Response {
	e.Deps.Notifier.Logf("[Group] %s → бот упомянут, отвечаю", ctx.SenderName)

	result, err := e.Deps.GPTService.ReplyFromGroupHistory(chat)
	e.Deps.Notifier.LogError(err)
	e.Deps.Notifier.Logf("[GroupGPT] %s", result.Text)

	return chatResultToResponses(result, chat.Settings.UseMarkdown)
}
