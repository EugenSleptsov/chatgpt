package executor

import (
	"GPTBot/application/service"
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

// TextExecutor is the catch-all executor for text messages.
// In private chats it runs GPT chat-completion; in groups it detects
// mentions, logs context, and decides whether to reply.
type TextExecutor struct {
	BotUsername             string
	GPT                     *service.GPTService
	Commands                *service.GPTCommandService
	AIClient                ai.Client
	History                 *service.HistoryService
	Notifier                *service.Notifier
	Auth                    *service.Auth
	DefaultAutoReplyPersona string // fallback persona from config; empty = built-in default
}

func (e *TextExecutor) Match(_ *pipeline.RequestContext) bool {
	return true // catch-all
}

func (e *TextExecutor) Execute(ctx *pipeline.RequestContext, c *chat.Chat) []sender.Response {
	return e.ProcessText(ctx, c, ctx.Text, false)
}

// ProcessText is the core text-handling logic.
// It is also called by VoiceExecutor for non-forwarded voice messages.
func (e *TextExecutor) ProcessText(ctx *pipeline.RequestContext, c *chat.Chat, text string, isVoice bool) []sender.Response {
	if text == "" {
		return nil
	}

	if !ctx.IsGroup {
		return e.privateChat(ctx, c, text, isVoice)
	}
	return e.groupChat(ctx, c, text, isVoice)
}

func (e *TextExecutor) privateChat(ctx *pipeline.RequestContext, c *chat.Chat, text string, isVoice bool) []sender.Response {
	session := c.ActiveSession()
	e.History.Append(session, chat.Message{Role: "user", Content: text}, c.Settings.MaxMessages)

	result, err := e.GPT.Complete(c)
	e.Notifier.LogError(err)

	if result.Text != "" {
		e.Notifier.Logf("[ChatGPT] %s", result.Text)
	}
	e.Notifier.ReportAdmin(ctx.SenderID, fmt.Sprintf(
		"[%s | %s]\nMessage: %s\nResponse: %s\nImages: %d, Audio: %v\n%s",
		c.Title, session.Model, text, result.Text,
		len(result.Images), result.Audio != nil, result.Usage,
	))

	responses := chatResultToResponses(result, c.Settings.UseMarkdown)

	// Voice-input guarantee: if the user sent voice and GPT didn't call
	// generate_voice, we synthesize audio from the text response.
	if isVoice && result.Audio == nil {
		audio, voiceErr := e.AIClient.GenerateVoice(result.Text, ai.VoiceModelHD, ai.VoiceOnyx)
		e.Notifier.LogError(voiceErr)
		if audio != nil {
			responses = append(responses, sender.Response{Audio: audio})
		}
	}

	return responses
}

func (e *TextExecutor) groupChat(ctx *pipeline.RequestContext, c *chat.Chat, text string, isVoice bool) []sender.Response {
	botMentioned := strings.Contains(text, "@"+e.BotUsername)
	botCalledByName := strings.Contains(strings.ToLower(text), "бот")
	isReplyToBot := ctx.ReplyToUsername == e.BotUsername
	botAddressed := botMentioned || isReplyToBot || botCalledByName

	cleanText := text
	if botMentioned {
		cleanText = strings.TrimSpace(strings.ReplaceAll(text, "@"+e.BotUsername, ""))
	}

	// Always log message for group context
	e.History.LogGroupMessage(c, ctx.SenderName, cleanText)

	// Edited messages: just updated context, nothing to process
	if ctx.IsEdited {
		e.Notifier.Logf("[Group] %s → отредактировано, обновляю контекст", ctx.SenderName)
		return nil
	}

	if botAddressed {
		e.Notifier.Logf("[Group] %s → бот упомянут, отвечаю", ctx.SenderName)
		return e.completeGroup(ctx, c, "Group reply")
	}

	if c.Settings.GroupAutoReply {
		if !e.Auth.IsAuthorized(ctx.SenderID) {
			return nil
		}
		persona := c.Settings.AutoReplyPersona
		if persona == "" {
			persona = e.DefaultAutoReplyPersona
		}
		should, reason, err := e.Commands.ShouldAutoReply(c, persona)
		e.Notifier.LogError(err)
		if !should {
			e.Notifier.Logf("[Group] Авто-ответ: НЕТ (%s)", reason)
			return nil
		}
		e.Notifier.Logf("[Group] Авто-ответ: ДА (%s)", reason)
		return e.completeGroup(ctx, c, "Group auto-reply")
	}

	return nil
}

// completeGroup runs GPT on the active session and reports results. It guards
// against empty history so callers don't have to.
func (e *TextExecutor) completeGroup(ctx *pipeline.RequestContext, c *chat.Chat, label string) []sender.Response {
	session := c.ActiveSession()
	if len(session.History) == 0 {
		return nil
	}
	result, err := e.GPT.Complete(c)
	e.Notifier.LogError(err)
	e.Notifier.Logf("[GroupGPT] %s | %s", result.Text, result.Usage.Summary())
	e.Notifier.ReportAdmin(ctx.SenderID, fmt.Sprintf(
		"[%s | %s]\n%s\nResponse: %s\n%s",
		c.Title, session.Model, label, result.Text, result.Usage,
	))
	return chatResultToResponses(result, c.Settings.UseMarkdown)
}

// chatResultToResponses maps a service.ChatResult to []pipeline.Response.
// When images are present the model's text reply becomes the caption of the
// first image instead of being sent as a separate message.
func chatResultToResponses(r *service.ChatResult, markdown bool) []sender.Response {
	var out []sender.Response

	textUsed := false
	for i, img := range r.Images {
		caption := ""
		if i == 0 && r.Text != "" {
			caption = r.Text
			textUsed = true
		}
		out = append(out, sender.Response{ImageData: img.Data, Caption: caption})
	}

	if r.Text != "" && !textUsed {
		out = append(out, sender.Response{Text: r.Text, Markdown: markdown})
	}
	if r.Audio != nil {
		out = append(out, sender.Response{Audio: r.Audio})
	}

	return out
}
