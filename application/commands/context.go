package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strings"
)

// CommandContext shows context window usage analysis for the active session.
type CommandContext struct {
	ContextWindowFn func(tierID string) int // returns max tokens for a model tier
}

func (c *CommandContext) Name() string { return "context" }
func (c *CommandContext) Description() string {
	return "Показать использование контекстного окна текущей сессии"
}
func (c *CommandContext) IsAdmin() bool { return false }

func (c *CommandContext) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	session := ch.ActiveSession()

	// Estimate token counts per section (~4 chars per token).
	systemTokens := len(session.SystemPrompt) / 4
	memoryTokens := 0
	for _, fact := range ch.Memory {
		memoryTokens += len(fact) / 4
	}

	historyTokens := 0
	historyMsgs := 0
	for _, entry := range session.History {
		historyTokens += len(entry.Prompt.Content) / 4
		historyMsgs++
		if entry.Response != (chat.Message{}) {
			historyTokens += len(entry.Response.Content) / 4
			historyMsgs++
		}
	}

	totalTokens := systemTokens + memoryTokens + historyTokens

	// Context window limit
	contextWindow := 128_000 // default
	if c.ContextWindowFn != nil {
		contextWindow = c.ContextWindowFn(session.Model)
	}
	usagePct := float64(totalTokens) / float64(contextWindow) * 100

	var sb strings.Builder
	sb.WriteString("📐 Контекстное окно\n\n")
	sb.WriteString(fmt.Sprintf("Модель: %s (лимит: %dk токенов)\n\n", session.Model, contextWindow/1000))

	// Breakdown
	sb.WriteString(fmt.Sprintf("Системный промпт: ~%d токенов\n", systemTokens))
	sb.WriteString(fmt.Sprintf("Память: ~%d токенов (%d фактов)\n", memoryTokens, len(ch.Memory)))
	sb.WriteString(fmt.Sprintf("История: ~%d токенов (%d сообщений)\n", historyTokens, historyMsgs))
	sb.WriteString(fmt.Sprintf("Итого: ~%d токенов\n\n", totalTokens))

	// Visual bar
	bar := progressBar(usagePct)
	sb.WriteString(fmt.Sprintf("%s %.1f%%\n", bar, usagePct))

	if usagePct > 75 {
		sb.WriteString("\n⚠️ Контекст заполнен более чем на 75%. Скоро произойдёт автоматическое сжатие.")
	} else if usagePct > 50 {
		sb.WriteString("\nℹ️ Контекст заполнен наполовину.")
	}

	return reply(sb.String())
}
