package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
	"time"
)

// CommandSettings is a self-contained button hub for per-chat settings.
// Every control routes back into this command via callback data "settings:<sub>"
// so the message is edited in place and the hub keeps its navigation context
// (unlike deep-linking to the standalone /model, /markdown … commands, which
// lose the "back" affordance). The standalone commands remain for typed use.
//
// Sub-routes:
//
//	""              → render the hub
//	"md"            → toggle Markdown, re-render hub
//	"ar"            → toggle group auto-reply (admin), re-render hub
//	"model"         → render model picker (back → hub)
//	"model:<id>"    → set model tier, re-render picker
//	"sm"            → toggle supermemory, re-render hub
//	"role"          → show auto-reply persona + edit hint (back → hub)
//	"sprompt"       → show summarize prompt + edit hint (back → hub)
type CommandSettings struct {
	Auth *service.Auth
}

func (c *CommandSettings) Name() string        { return "settings" }
func (c *CommandSettings) Description() string { return "Настройки чата (кнопки)." }
func (c *CommandSettings) IsAdmin() bool       { return false }

func (c *CommandSettings) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	args := strings.TrimSpace(ctx.CommandArgs)
	isAdmin := c.Auth.IsAdmin(ctx.SenderID)

	switch {
	case args == "md":
		ch.Settings.UseMarkdown = !ch.Settings.UseMarkdown
	case args == "delconfirm":
		ch.Settings.SkipDeleteConfirm = !ch.Settings.SkipDeleteConfirm
	case args == "ar" && isAdmin:
		ch.Settings.GroupAutoReply = !ch.Settings.GroupAutoReply
	case args == "verbose" && isAdmin:
		ch.Settings.Verbose = !ch.Settings.Verbose
	case args == "model":
		return settingsModelView(ch)
	case strings.HasPrefix(args, "model:"):
		if t := ai.FindTier(args[len("model:"):]); t != nil {
			ch.ActiveSession().Model = t.ID
		}
		return settingsModelView(ch)
	case args == "sm":
		ch.Settings.Supermemory = !ch.Settings.Supermemory

	case args == "tz":
		return settingsTimezoneView(ch)
	case strings.HasPrefix(args, "tz:set:"):
		if zone := args[len("tz:set:"):]; zone != "" {
			if _, err := time.LoadLocation(zone); err == nil {
				ch.Settings.Timezone = zone
			}
		}
		return settingsTimezoneView(ch)
	case args == "tz:edit":
		ch.PendingInput = "timezone"
		return forceReplyPrompt("Пришлите часовой пояс (IANA, напр. Europe/Berlin):")

	case args == "system":
		sp := ch.ActiveSession().SystemPrompt
		if sp == "" {
			sp = "(не задан)"
		}
		return editView("Системный промпт:\n\n"+sp, "settings:system:edit")
	case args == "system:edit":
		ch.PendingInput = "system"
		return forceReplyPrompt("Пришлите новый системный промпт:")

	case args == "sprompt":
		sp := ch.Settings.SummarizePrompt
		if sp == "" {
			sp = "(по умолчанию)"
		}
		return editView("Промпт суммаризации:\n\n"+sp, "settings:sprompt:edit")
	case args == "sprompt:edit":
		ch.PendingInput = "summarize_prompt"
		return forceReplyPrompt("Пришлите новый промпт суммаризации:")

	case args == "role" && isAdmin:
		persona := ch.Settings.AutoReplyPersona
		if persona == "" {
			persona = service.DefaultAutoReplyPersona
		}
		return editView("Роль авто-ответа:\n\n"+persona, "settings:role:edit")
	case args == "role:edit" && isAdmin:
		ch.PendingInput = "autorole"
		return forceReplyPrompt("Пришлите новый текст роли авто-ответа:")
	}

	return settingsHubView(ch, isAdmin)
}

// onOff renders a boolean as a check/cross marker.
func onOff(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}

// settingsHubView renders the top-level settings keyboard. Toggle buttons flip
// state on tap (single button, not on/off pair); the rest drill into sub-views.
func settingsHubView(ch *chat.Chat, isAdmin bool) []sender.Response {
	modelLabel := ch.ActiveSession().Model
	if t := ai.FindTier(modelLabel); t != nil {
		modelLabel = t.Label
	}

	rows := [][]sender.Button{
		{{Text: "Модель: " + modelLabel, Data: "settings:model"}},
		{{Text: "Markdown " + onOff(ch.Settings.UseMarkdown), Data: "settings:md"}},
		{{Text: "Удалять сессии без подтверждения " + onOff(ch.Settings.SkipDeleteConfirm), Data: "settings:delconfirm"}},
		{{Text: "📝 Системный промпт", Data: "settings:system"}},
		{{Text: "Промпт суммаризации", Data: "settings:sprompt"}},
		{{Text: "🧠 Суперпамять " + onOff(ch.Settings.Supermemory), Data: "settings:sm"}},
		{{Text: "🌍 Часовой пояс: " + timezoneLabel(ch), Data: "settings:tz"}},
	}
	if isAdmin {
		rows = append(rows,
			[]sender.Button{{Text: "Авто-ответ " + onOff(ch.Settings.GroupAutoReply), Data: "settings:ar"}},
			[]sender.Button{{Text: "Роль авто-ответа", Data: "settings:role"}},
			[]sender.Button{{Text: "Вербозность " + onOff(ch.Settings.Verbose), Data: "settings:verbose"}},
		)
	}
	rows = append(rows, []sender.Button{{Text: "⬅ Меню", Data: "menu:"}})

	return []sender.Response{{Text: "⚙️ Настройки чата", Buttons: rows}}
}

// settingsModelView renders the tier picker with a back row to the hub.
func settingsModelView(ch *chat.Chat) []sender.Response {
	current := ch.ActiveSession().Model
	row := make([]sender.Button, 0, len(ai.Tiers))
	for _, t := range ai.Tiers {
		label := t.Label
		if t.ID == current {
			label = "✅ " + label
		}
		row = append(row, sender.Button{Text: label, Data: "settings:model:" + t.ID})
	}
	return []sender.Response{{
		Text:    "Выберите модель:",
		Buttons: [][]sender.Button{row, backRow()},
	}}
}

// settingsTimezonePresets are the one-tap zone choices in the timezone view.
var settingsTimezonePresets = []struct{ Label, Zone string }{
	{"Берлин", "Europe/Berlin"},
	{"Москва", "Europe/Moscow"},
	{"Киев", "Europe/Kyiv"},
	{"UTC", "UTC"},
}

// settingsTimezoneView renders the timezone picker: current zone with local
// time, preset buttons and a manual-input fallback.
func settingsTimezoneView(ch *chat.Chat) []sender.Response {
	rows := make([][]sender.Button, 0, len(settingsTimezonePresets)+2)
	for _, p := range settingsTimezonePresets {
		label := p.Label
		if ch.Settings.Timezone == p.Zone {
			label = "✅ " + label
		}
		rows = append(rows, []sender.Button{{Text: label, Data: "settings:tz:set:" + p.Zone}})
	}
	rows = append(rows,
		[]sender.Button{{Text: "✏️ Ввести вручную", Data: "settings:tz:edit"}},
		backRow(),
	)
	return []sender.Response{{
		Text: fmt.Sprintf("🌍 Часовой пояс: %s\nСейчас: %s\n\nИспользуется для напоминаний и отображения времени.",
			timezoneLabel(ch), time.Now().In(ch.Location()).Format("02.01 15:04")),
		Buttons: rows,
	}}
}

// backRow is a single "back to settings hub" button row.
func backRow() []sender.Button {
	return []sender.Button{{Text: "⬅ Назад", Data: "settings:"}}
}

// editView shows current value text with an "edit" button (which starts a
// ForceReply input flow via editData) and a back-to-hub row.
func editView(text, editData string) []sender.Response {
	return []sender.Response{{
		Text: text,
		Buttons: [][]sender.Button{
			{{Text: "✏️ Изменить", Data: editData}},
			backRow(),
		},
	}}
}

// forceReplyPrompt returns a message that opens a Telegram reply box; the user's
// reply is captured by the worker and routed to the pending command.
func forceReplyPrompt(text string) []sender.Response {
	return []sender.Response{{Text: text, ForceReply: true}}
}
