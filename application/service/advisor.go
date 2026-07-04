package service

import (
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
)

// Advisor helpers. Advisor notes belong to the chat (shared across sessions,
// like Memory): the model files each note under a topic it names itself
// ("Налоги", "Покупки в ИКЕА", ...). Plain functions over the domain type.

const (
	maxAdvisorTopicLen = 64
	maxAdvisorNoteLen  = 512
)

// AddAdvisorNote validates and stores one note under the given topic,
// creating the topic when needed. Returns the topic the note landed in.
func AddAdvisorNote(chat *chatdomain.Chat, topic, note string) (*chatdomain.AdvisorTopic, error) {
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
	t, _ := chat.AddAdvisorNote(topic, note)
	log.Printf("[Advisor] note added to %q: %s (topic now %d entries)", t.Name, note, len(t.Entries))
	return t, nil
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
// read_notes tool output.
func AdvisorNotesForTool(topic *chatdomain.AdvisorTopic) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic %q, %d entries:\n", topic.Name, len(topic.Entries)))
	for _, e := range topic.Entries {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.Created.Format("2006-01-02"), e.Text))
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
