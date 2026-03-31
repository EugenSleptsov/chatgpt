package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
	"time"
)

// CommandUsage shows cumulative cost and token statistics for the current chat.
type CommandUsage struct{}

func (c *CommandUsage) Name() string { return "usage" }
func (c *CommandUsage) Description() string {
	return "Показать статистику расходов текущего чата за сегодня"
}
func (c *CommandUsage) IsAdmin() bool { return false }

func (c *CommandUsage) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	var sb strings.Builder

	sb.WriteString("📊 Статистика за сегодня\n\n")

	if ch.TotalRequests == 0 || !sameDay(ch.CostResetTime, time.Now()) {
		sb.WriteString("Нет запросов за сегодня.")
		return reply(sb.String())
	}

	sb.WriteString(fmt.Sprintf("Запросов: %d\n", ch.TotalRequests))
	sb.WriteString(fmt.Sprintf("Входные токены: %d\n", ch.TotalInputTokens))
	sb.WriteString(fmt.Sprintf("Выходные токены: %d\n", ch.TotalOutputTokens))
	sb.WriteString(fmt.Sprintf("Стоимость: $%.4f\n", ch.TotalCostUSD))

	if ch.Settings.CostLimitUSD > 0 {
		pct := ch.TotalCostUSD / ch.Settings.CostLimitUSD * 100
		bar := progressBar(pct)
		sb.WriteString(fmt.Sprintf("\nЛимит: $%.2f\n%s %.1f%%", ch.Settings.CostLimitUSD, bar, pct))
	}

	// Session info
	session := ch.ActiveSession()
	sb.WriteString(fmt.Sprintf("\n\nСессия: %s (модель: %s, сообщений: %d)",
		session.Topic, session.Model, len(session.History)))

	return reply(sb.String())
}

// progressBar renders a text progress bar (10 segments).
func progressBar(pct float64) string {
	const total = 10
	filled := int(pct / 100 * total)
	if filled > total {
		filled = total
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", total-filled) + "]"
}

// sameDay checks if two times are on the same calendar day.
func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
