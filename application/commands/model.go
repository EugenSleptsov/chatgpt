package commands

import (
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
)

type CommandModel struct{}

func (c *CommandModel) Name() string {
	return "model"
}

func (c *CommandModel) Description() string {
	return "Показывает или устанавливает модель. Использование: /model [ID]"
}

func (c *CommandModel) IsAdmin() bool {
	return false
}

func (c *CommandModel) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	session := chat.ActiveSession()
	args := ctx.CommandArgs

	// An argument (typed "/model premium" or a button tap "model:premium")
	// selects a tier; no argument just shows the current selection.
	if args != "" {
		tier := ai.FindTier(args)
		if tier == nil {
			return modelView(session.Model, fmt.Sprintf("Модель не найдена: %s", args))
		}
		session.Model = tier.ID
	}

	return modelView(session.Model, "")
}

// modelView renders the model picker: a header line plus a row of buttons (one
// per tier, the active one marked). An optional notice is prepended. Tapping a
// button sends callback data "model:<tierID>", routed back into Execute.
func modelView(current, notice string) []sender.Response {
	tier := ai.FindTier(current)
	name := current
	if tier != nil {
		name = tier.Label
	}

	header := fmt.Sprintf("Текущая модель: %s\n\nВыберите модель:", name)
	if notice != "" {
		header = notice + "\n\n" + header
	}

	row := make([]sender.Button, 0, len(ai.Tiers))
	for _, t := range ai.Tiers {
		label := t.Label
		if t.ID == current {
			label = "✅ " + label
		}
		row = append(row, sender.Button{Text: label, Data: "model:" + t.ID})
	}

	return []sender.Response{{Text: header, Buttons: [][]sender.Button{row}}}
}
