package telegram

import (
	"strings"
)

const maxMessageLen = 4096
const maxChunks = 10

// markdownToHTML converts GPT's CommonMark output to Telegram HTML.
// Handles: **bold**, __underline__, _italic_, `code`, ```code blocks```, [text](url).
// Plain text is HTML-escaped. Falls through gracefully on unrecognised patterns.
func markdownToHTML(input string) string {
	var b strings.Builder
	convertMD([]rune(input), &b)
	return b.String()
}

func convertMD(runes []rune, b *strings.Builder) {
	n := len(runes)
	i := 0
	for i < n {
		// ``` code block ```
		if mdMatch(runes, i, "```") {
			if r_close := mdFind(runes, i+3, "```"); r_close >= 0 {
				content := string(runes[i+3 : r_close])
				// strip optional language tag on first line (e.g. ```python\n...)
				if nl := strings.IndexByte(content, '\n'); nl >= 0 {
					content = content[nl+1:]
				}
				b.WriteString("<pre><code>")
				htmlEscape(b, content)
				b.WriteString("</code></pre>")
				i = r_close + 3
				continue
			}
		}

		// `inline code`
		if runes[i] == '`' {
			if r_close := mdFindRune(runes, i+1, '`'); r_close >= 0 {
				b.WriteString("<code>")
				htmlEscape(b, string(runes[i+1:r_close]))
				b.WriteString("</code>")
				i = r_close + 1
				continue
			}
		}

		// **bold** — check before single *
		if mdMatch(runes, i, "**") {
			if r_close := mdFind(runes, i+2, "**"); r_close >= 0 && !mdHasNewline(runes, i+2, r_close) {
				b.WriteString("<b>")
				convertMD(runes[i+2:r_close], b)
				b.WriteString("</b>")
				i = r_close + 2
				continue
			}
		}

		// __underline__ — check before single _
		if mdMatch(runes, i, "__") {
			if r_close := mdFind(runes, i+2, "__"); r_close >= 0 && !mdHasNewline(runes, i+2, r_close) {
				b.WriteString("<u>")
				convertMD(runes[i+2:r_close], b)
				b.WriteString("</u>")
				i = r_close + 2
				continue
			}
		}

		// _italic_ — only at word boundaries to avoid snake_case false positives
		if runes[i] == '_' && mdAtBoundaryLeft(runes, i) {
			if r_close := mdFindRune(runes, i+1, '_'); r_close >= 0 &&
				mdAtBoundaryRight(runes, r_close) &&
				!mdHasNewline(runes, i+1, r_close) {
				b.WriteString("<i>")
				convertMD(runes[i+1:r_close], b)
				b.WriteString("</i>")
				i = r_close + 1
				continue
			}
		}

		// *italic* — not followed by space (to avoid list bullets: "* item")
		if runes[i] == '*' && i+1 < n && runes[i+1] != ' ' && runes[i+1] != '\n' {
			if r_close := mdFindRune(runes, i+1, '*'); r_close >= 0 && !mdHasNewline(runes, i+1, r_close) {
				b.WriteString("<i>")
				convertMD(runes[i+1:r_close], b)
				b.WriteString("</i>")
				i = r_close + 1
				continue
			}
		}

		// [text](url) link
		if runes[i] == '[' {
			if closeText := mdFindRune(runes, i+1, ']'); closeText >= 0 &&
				closeText+1 < n && runes[closeText+1] == '(' {
				if closeURL := mdFindRune(runes, closeText+2, ')'); closeURL >= 0 {
					url := string(runes[closeText+2 : closeURL])
					b.WriteString(`<a href="`)
					htmlEscape(b, url)
					b.WriteString(`">`)
					convertMD(runes[i+1:closeText], b)
					b.WriteString("</a>")
					i = closeURL + 1
					continue
				}
			}
		}

		// plain character — HTML-escape only
		htmlEscapeRune(b, runes[i])
		i++
	}
}

// --- HTML escaping ---

func htmlEscape(b *strings.Builder, s string) {
	for _, r := range s {
		htmlEscapeRune(b, r)
	}
}

func htmlEscapeRune(b *strings.Builder, r rune) {
	switch r {
	case '&':
		b.WriteString("&amp;")
	case '<':
		b.WriteString("&lt;")
	case '>':
		b.WriteString("&gt;")
	case '"':
		b.WriteString("&quot;")
	default:
		b.WriteRune(r)
	}
}

// --- Markdown parsing helpers ---

func mdMatch(runes []rune, i int, s string) bool {
	sub := []rune(s)
	if i+len(sub) > len(runes) {
		return false
	}
	for j, r := range sub {
		if runes[i+j] != r {
			return false
		}
	}
	return true
}

func mdFind(runes []rune, from int, substr string) int {
	sub := []rune(substr)
	l := len(sub)
	for i := from; i <= len(runes)-l; i++ {
		ok := true
		for j := 0; j < l; j++ {
			if runes[i+j] != sub[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func mdFindRune(runes []rune, from int, r rune) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == r {
			return i
		}
	}
	return -1
}

func mdHasNewline(runes []rune, from, to int) bool {
	for i := from; i < to && i < len(runes); i++ {
		if runes[i] == '\n' {
			return true
		}
	}
	return false
}

func mdAtBoundaryLeft(runes []rune, i int) bool {
	if i == 0 {
		return true
	}
	r := runes[i-1]
	return r == ' ' || r == '\t' || r == '\n' || r == '(' || r == '['
}

func mdAtBoundaryRight(runes []rune, i int) bool {
	if i+1 >= len(runes) {
		return true
	}
	r := runes[i+1]
	return r == ' ' || r == '\t' || r == '\n' || r == ')' || r == ']' || r == ',' || r == '.'
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

		if end < len(runes) {
			if idx := lastNewline(runes[:end]); idx > 0 {
				end = idx + 1
			}
		}

		chunk := string(runes[:end])
		runes = runes[end:]

		fences := strings.Count(chunk, "```")

		if codeBlockOpen {
			chunk = "```\n" + chunk
			fences++
		}

		codeBlockOpen = (fences % 2) == 1
		if codeBlockOpen {
			chunk += "\n```"
		}

		chunks = append(chunks, chunk)

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
