package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"strings"
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
//	"memory"        → show memory (back → hub)
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
	case args == "ar" && isAdmin:
		ch.Settings.GroupAutoReply = !ch.Settings.GroupAutoReply
	case args == "model":
		return settingsModelView(ch)
	case strings.HasPrefix(args, "model:"):
		if t := ai.FindTier(args[len("model:"):]); t != nil {
			ch.ActiveSession().Model = t.ID
		}
		return settingsModelView(ch)
	case args == "memory":
		return backView(service.FormatMemory(ch))

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
		ch.PendingInput = "summarizeprompt"
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
		{{Text: "📝 Системный промпт", Data: "settings:system"}},
		{{Text: "Промпт суммаризации", Data: "settings:sprompt"}},
		{{Text: "🧠 Память", Data: "settings:memory"}},
	}
	if isAdmin {
		rows = append(rows,
			[]sender.Button{{Text: "Авто-ответ " + onOff(ch.Settings.GroupAutoReply), Data: "settings:ar"}},
			[]sender.Button{{Text: "Роль авто-ответа", Data: "settings:role"}},
		)
	}

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

// backRow is a single "back to settings hub" button row.
func backRow() []sender.Button {
	return []sender.Button{{Text: "⬅ Назад", Data: "settings:"}}
}

// backView wraps text with a single back-to-hub row.
func backView(text string) []sender.Response {
	return []sender.Response{{Text: text, Buttons: [][]sender.Button{backRow()}}}
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
