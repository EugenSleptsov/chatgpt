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

const historyPageSize = 5

type CommandHistory struct{}

func (c *CommandHistory) Name() string {
	return "history"
}

func (c *CommandHistory) Description() string {
	return "Показывает историю разговоров. /history [страница]"
}

func (c *CommandHistory) IsAdmin() bool {
	return false
}

func (c *CommandHistory) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	// decode page number (default = 1 = latest)
	page := 1
	if arg := strings.TrimSpace(ctx.CommandArgs); arg != "" {
		if p, err := strconv.Atoi(arg); err == nil && p > 0 {
			page = p
		}
	}

	pageChunks, totalPages := service.FormatHistoryPage(chat.ActiveSession(), page, historyPageSize)

	var responses []sender.Response
	for _, message := range pageChunks {
		responses = append(responses, sender.Response{Text: message})
	}

	if len(responses) == 0 {
		responses = reply("История пуста.")
	}

	// navigation hint
	if totalPages > 1 {
		// Recalculate effective page for hint (FormatPage clamps it)
		effectivePage := page
		if effectivePage > totalPages {
			effectivePage = totalPages
		}
		hint := fmt.Sprintf("📄 Страница %d из %d.", effectivePage, totalPages)
		if effectivePage < totalPages {
			hint += fmt.Sprintf(" Ранние сообщения: /history %d", effectivePage+1)
		}
		responses = append(responses, sender.Response{Text: hint})
	}

	return withInfoBack(ctx, responses)
}
