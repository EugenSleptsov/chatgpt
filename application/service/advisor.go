package service

import (
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
	"time"
)

// Advisor helpers. Advisor notes belong to the chat (shared across sessions,
// like Memory): the model files each note under a topic it names itself
// ("Налоги", "Покупки в ИКЕА", ...). Plain functions over the domain type.

const (
	maxAdvisorTopicLen = 64
	maxAdvisorNoteLen  = 512

	// defaultRemindHour is used when a reminder comes with a date but no time.
	defaultRemindHour = 9
)

// AddAdvisorNote validates and stores one note under the given topic,
// creating the topic when needed. remindAt is optional (nil = plain note).
// Returns the topic the note landed in.
func AddAdvisorNote(chat *chatdomain.Chat, topic, note string, remindAt *time.Time) (*chatdomain.AdvisorTopic, error) {
	topic = strings.TrimSpace(topic)
	note = strings.TrimSpace(note)
	if topic == "" || note == "" {
		return nil, fmt.Errorf("topic and note must be non-empty")
	}
	topic = truncateRunes(topic, maxAdvisorTopicLen)
	note = truncateRunes(note, maxAdvisorNoteLen)
	t, entry := chat.AddAdvisorNote(topic, note)
	entry.RemindAt = remindAt
	log.Printf("[Advisor] note added to %q: %s (topic now %d entries, remind: %v)", t.Name, note, len(t.Entries), remindAt)
	return t, nil
}

// ParseRemindAt parses a reminder time coming from the model, interpreted in
// the chat's timezone: "2006-01-02 15:04" (also with a 'T' separator) or a
// bare date "2006-01-02", which defaults to defaultRemindHour. Empty input
// means "no reminder" and yields nil without error. Times already in the past
// (relative to now, with a minute of grace) are rejected so a model mistake
// doesn't fire a reminder instantly.
func ParseRemindAt(s string, loc *time.Location, now time.Time) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if loc == nil {
		loc = time.Local
	}
	var t time.Time
	parsed := false
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04"} {
		if p, err := time.ParseInLocation(layout, s, loc); err == nil {
			t, parsed = p, true
			break
		}
	}
	if !parsed {
		if d, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
			t = time.Date(d.Year(), d.Month(), d.Day(), defaultRemindHour, 0, 0, 0, loc)
			parsed = true
		}
	}
	if !parsed {
		return nil, fmt.Errorf("invalid remind_at %q, expected YYYY-MM-DD or YYYY-MM-DD HH:MM", s)
	}
	if t.Before(now.Add(-time.Minute)) {
		return nil, fmt.Errorf("remind_at %q is in the past (now is %s) — use a future time", s, now.In(loc).Format("2006-01-02 15:04"))
	}
	return &t, nil
}

// truncateRunes shortens s to max runes (byte slicing would cut multi-byte
// characters — Cyrillic topics — in half).
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// AdvisorPrompt returns the advisor section for the system prompt: topic
// names with entry counts only — full entries are fetched via the read_notes
// tool to keep the prompt small.
func AdvisorPrompt(chat *chatdomain.Chat) string {
	if len(chat.AdvisorTopics) == 0 {
		return ""
	}
	names := make([]string, 0, len(chat.AdvisorTopics))
	for _, t := range chat.AdvisorTopics {
		names = append(names, fmt.Sprintf("%s (%d)", t.Name, len(t.Entries)))
	}
	return "Advisor note topics (use read_notes to see entries): " + strings.Join(names, ", ")
}

// AdvisorNotesForTool renders one topic's entries as plain text for the
// read_notes tool output. Entry IDs are included so the model can reference
// them in set_reminder. Times are rendered in the chat's timezone so the model
// and the user talk about the same wall clock.
func AdvisorNotesForTool(topic *chatdomain.AdvisorTopic, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic %q, %d entries:\n", topic.Name, len(topic.Entries)))
	for _, e := range topic.Entries {
		sb.WriteString(fmt.Sprintf("- #%d [%s] %s", e.ID, e.Created.In(loc).Format("2006-01-02"), e.Text))
		if e.RemindAt != nil {
			sb.WriteString(fmt.Sprintf(" (reminder: %s)", e.RemindAt.In(loc).Format("2006-01-02 15:04")))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// JoinPrompts joins non-empty prompt sections with a blank line. Used to
// combine memory and advisor sections into the single "memory prompt" string
// threaded through instructions, compaction and token metrics.
func JoinPrompts(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, "\n\n")
}
