package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
)

// CommandDream reorganizes supermemory on demand: the model scans the root
// node index and folds scattered fragments, duplicated topics and clutter into
// thematic parent nodes (lossless — children stay readable below the parent).
// The call is slow, so a transient "бот заснул" status is shown while it runs.
type CommandDream struct {
	Commands *service.GPTService
	Progress service.ProgressReporter // "заснул" status while dreaming (may be nil)
}

func (c *CommandDream) Name() string { return "dream" }
func (c *CommandDream) Description() string {
	return "Сон: реорганизация памяти."
}
func (c *CommandDream) IsAdmin() bool { return false }

func (c *CommandDream) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	back := [][]sender.Button{{
		{Text: "🧠 Память", Data: "memory:"},
		{Text: "⬅ Меню", Data: "menu:"},
	}}
	if !ch.Settings.Supermemory {
		return []sender.Response{{
			Text:    "🧠 Суперпамять выключена — сон ни к чему. Включить можно в /memory.",
			Buttons: back,
		}}
	}

	done := service.StartProgress(c.Progress, ch.ChatID, "💤 Бот заснул — перебирает память…")
	defer done()
	report, _ := c.Commands.Dream(ch)

	return []sender.Response{{
		Text:    "⏰ Бот проснулся.\n\n" + report,
		Buttons: back,
	}}
}
