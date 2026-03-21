package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strconv"
	"strings"
)

const historyPageSize = 5

type CommandHistory struct {
	*Deps
}

func (c *CommandHistory) Name() string {
	return "history"
}

func (c *CommandHistory) Description() string {
	return "Показывает историю разговоров. /history [страница]"
}

func (c *CommandHistory) IsAdmin() bool {
	return false
}

func (c *CommandHistory) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	chunks := formatHistory(storage.ToGPTMessages(chat.ActiveSession().History))
	totalPages := (len(chunks) + historyPageSize - 1) / historyPageSize

	// parse page number (default = 1 = latest)
	page := 1
	if arg := strings.TrimSpace(ctx.Msg.CommandArguments()); arg != "" {
		if p, err := strconv.Atoi(arg); err == nil && p > 0 {
			page = p
		}
	}

	if page > totalPages {
		page = totalPages
	}

	// page 1 = last N chunks, page 2 = previous N, etc.
	end := len(chunks) - (page-1)*historyPageSize
	start := end - historyPageSize
	if start < 0 {
		start = 0
	}

	pageChunks := chunks[start:end]

	var responses []handler.Response
	for _, message := range pageChunks {
		responses = append(responses, handler.Response{Text: message})
	}

	// navigation hint
	if totalPages > 1 {
		hint := fmt.Sprintf("📄 Страница %d из %d.", page, totalPages)
		if page < totalPages {
			hint += fmt.Sprintf(" Ранние сообщения: /history %d", page+1)
		}
		responses = append(responses, handler.Response{Text: hint})
	}

	return responses
}

func formatHistory(history []gpt.Message) []string {
	if len(history) == 0 {
		return []string{"История разговоров пуста."}
	}

	var current string
	var chunks []string
	currentLen := 0

	for i, message := range history {
		line := fmt.Sprintf("%d. %s: %s\n", i+1, util.Title(message.Role), message.Content)
		lineRunes := len([]rune(line))

		if currentLen+lineRunes > 4096 {
			chunks = append(chunks, current)
			current = ""
			currentLen = 0
		}

		current += line
		currentLen += lineRunes
	}

	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}
