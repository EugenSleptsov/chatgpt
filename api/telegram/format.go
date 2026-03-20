package telegram

import (
	"GPTBot/util"
	"strings"
)

const maxMessageLen = 4096
const maxChunks = 10

// formatMarkdownV2 escapes text for Telegram MarkdownV2 and ensures code blocks are closed.
func formatMarkdownV2(text string) string {
	return util.FixMarkdown(escapeMarkdownV2(text))
}

// escapeMarkdownV2 escapes all special MarkdownV2 characters.
func escapeMarkdownV2(text string) string {
	charsToEscape := []string{"_", "*", "[", "]", "(", ")", "~", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range charsToEscape {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

// splitMessage splits text into chunks of at most maxMessageLen runes,
// preferring newline boundaries. Open code blocks (```) are properly
// closed/reopened across chunk boundaries. At most maxChunks are returned;
// excess text is truncated with an indicator.
func splitMessage(text string) []string {
	runes := []rune(text)
	if len(runes) <= maxMessageLen {
		return []string{text}
	}

	var chunks []string
	codeBlockOpen := false

	for len(runes) > 0 {
		end := maxMessageLen
		if end > len(runes) {
			end = len(runes)
		}

		// try to split at last newline within the limit
		if end < len(runes) {
			if idx := lastNewline(runes[:end]); idx > 0 {
				end = idx + 1
			}
		}

		chunk := string(runes[:end])
		runes = runes[end:]

		// count triple backticks in this chunk to track code block state
		fences := strings.Count(chunk, "```")

		if codeBlockOpen {
			chunk = "```\n" + chunk // reopen block from previous chunk
			fences++                // account for the added fence
		}

		// after this chunk, is a code block still open?
		codeBlockOpen = (fences % 2) == 1
		if codeBlockOpen {
			chunk += "\n```" // close dangling block for this chunk
		}

		chunks = append(chunks, chunk)

		// stop if we've reached the chunk limit
		if len(chunks) >= maxChunks && len(runes) > 0 {
			chunks[len(chunks)-1] += "\n\n... (сообщение обрезано)"
			break
		}
	}

	return chunks
}

func lastNewline(runes []rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '\n' {
			return i
		}
	}
	return -1
}
