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
	if len(topic) > maxAdvisorTopicLen {
		topic = topic[:maxAdvisorTopicLen]
	}
	if len(note) > maxAdvisorNoteLen {
		note = note[:maxAdvisorNoteLen]
	}
	t, entry := chat.AddAdvisorNote(topic, note)
	entry.RemindAt = remindAt
	log.Printf("[Advisor] note added to %q: %s (topic now %d entries, remind: %v)", t.Name, note, len(t.Entries), remindAt)
	return t, nil
}

// ParseRemindAt parses a reminder time coming from the model:
// "2006-01-02 15:04" (also with a 'T' separator) or a bare date "2006-01-02",
// which defaults to defaultRemindHour local time. Empty input means "no
// reminder" and yields nil without error.
func ParseRemindAt(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, nil
		}
	}
	if d, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		t := time.Date(d.Year(), d.Month(), d.Day(), defaultRemindHour, 0, 0, 0, time.Local)
		return &t, nil
	}
	return nil, fmt.Errorf("invalid remind_at %q, expected YYYY-MM-DD or YYYY-MM-DD HH:MM", s)
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
// them in set_reminder.
func AdvisorNotesForTool(topic *chatdomain.AdvisorTopic) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic %q, %d entries:\n", topic.Name, len(topic.Entries)))
	for _, e := range topic.Entries {
		sb.WriteString(fmt.Sprintf("- #%d [%s] %s", e.ID, e.Created.Format("2006-01-02"), e.Text))
		if e.RemindAt != nil {
			sb.WriteString(fmt.Sprintf(" (reminder: %s)", e.RemindAt.Format("2006-01-02 15:04")))
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
