package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
	"strings"
)

const sessionTransferPrompt = `Summarize the source conversation for transfer into a new conversation. Preserve formatting rules, agreements, formulas, important facts, decisions, and unfinished tasks. Clearly distinguish facts from assumptions. Do not invent anything. Return only the compact transferable context.`

type CommandSessionNew struct {
	Commands *service.GPTService
	Notifier *service.Notifier
	Progress service.ProgressReporter
}

func (c *CommandSessionNew) Name() string        { return "new" }
func (c *CommandSessionNew) Description() string { return "Создаёт новую сессию." }
func (c *CommandSessionNew) IsAdmin() bool       { return false }

func (c *CommandSessionNew) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	args := strings.TrimSpace(ctx.CommandArgs)
	if args == "" {
		return sessionNewModeView()
	}
	if ctx.IsCallback && strings.HasPrefix(args, "ask:") {
		return c.beginInput(ch, args)
	}

	mode, sourceID, topic := parseNewSessionArgs(args)
	// A typed /new <topic> remains a quick clean-session shortcut.
	if mode == "" {
		mode, topic = "clean", args
	}
	topic = sanitizeTopic(topic)
	if topic == "" {
		return sessionNewModeView()
	}

	source := ch.FindSession(sourceID)
	if source == nil {
		source = ch.ActiveSession()
	}

	var created *chat.Session
	switch mode {
	case "copy":
		created = ch.AddSessionFrom(topic, source, true)
	case "summary":
		if len(source.History) == 0 {
			created = ch.AddSessionFrom(topic, source, false)
			break
		}
		history := chat.ToGPTMessages(source.History)
		var text strings.Builder
		for _, message := range history {
			fmt.Fprintf(&text, "%s: %v\n", message.Role, message.Content)
		}
		done := service.StartProgress(c.Progress, ch.ChatID, "✨ Переношу контекст в новую сессию…")
		summary, usage, err := c.Commands.GPTCommand(source.Model, sessionTransferPrompt, text.String())
		done()
		if err != nil {
			if c.Notifier != nil {
				c.Notifier.LogError(err)
			}
			return []sender.Response{{
				Text: "Не удалось подготовить переносимый контекст. Попробовать ещё раз или создать чистую сессию?",
				Buttons: [][]sender.Button{
					{{Text: "🔄 Повторить", Data: fmt.Sprintf("new:ask:summary:%d", source.ID)}},
					{{Text: "🆕 Создать чистую", Data: fmt.Sprintf("new:ask:clean:%d", source.ID)}},
					{{Text: "⬅ Сессии", Data: "list:"}},
				},
			}}
		}
		ch.AccumulateCost(usage.Cost, usage.InputTokens, usage.OutputTokens)
		created = ch.AddSessionFrom(topic, source, false)
		created.TransferContext = summary
	default:
		created = ch.AddSessionFrom(topic, source, false)
		created.SystemPrompt = ""
		created.TransferContext = ""
	}
	ch.ActiveSessionID = created.ID
	return sessionListView(ch, sessionPageOf(ch, created.ID))
}

func sessionNewModeView() []sender.Response {
	return []sender.Response{{
		Text: "Как создать новую сессию?\n\n✨ Перенос — компактное резюме текущего контекста.\n📋 Копия — вся история целиком и может занять много контекста.\n🆕 Чистая — без истории.",
		Buttons: [][]sender.Button{
			{{Text: "✨ С переносом контекста", Data: "new:ask:summary"}},
			{{Text: "📋 Полная копия", Data: "new:ask:copy"}},
			{{Text: "🆕 Чистая", Data: "new:ask:clean"}},
			{{Text: "⬅ Сессии", Data: "list:"}},
		},
	}}
}

// Callback new:ask:<mode>[:source] starts the ForceReply step.
func (c *CommandSessionNew) beginInput(ch *chat.Chat, args string) []sender.Response {
	parts := strings.Split(args, ":")
	if len(parts) < 2 || parts[0] != "ask" {
		return sessionNewModeView()
	}
	sourceID := ch.ActiveSessionID
	if len(parts) > 2 {
		if id, err := strconv.Atoi(parts[2]); err == nil {
			sourceID = id
		}
	}
	ch.PendingInput = "new"
	ch.PendingInputArgs = fmt.Sprintf("%s:%d", parts[1], sourceID)
	return forceReplyPrompt("Название новой сессии:")
}

func parseNewSessionArgs(args string) (mode string, sourceID int, topic string) {
	lines := strings.SplitN(args, "\n", 2)
	if len(lines) != 2 {
		return "", 0, args
	}
	parts := strings.SplitN(lines[0], ":", 2)
	if len(parts) != 2 {
		return "", 0, args
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, args
	}
	return parts[0], id, lines[1]
}

func sanitizeTopic(raw string) string {
	topic := strings.TrimSpace(raw)
	runes := []rune(topic)
	if len(runes) > 64 {
		topic = string(runes[:64])
	}
	return topic
}
