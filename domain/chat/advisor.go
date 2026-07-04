package chat

import (
	"strings"
	"time"
)

// AdvisorEntry is a single saved note inside a topic.
type AdvisorEntry struct {
	ID       int
	Text     string
	Created  time.Time
	RemindAt *time.Time `json:",omitempty"` // when set, a reminder fires at this time
}

// AdvisorTopic groups advisor entries under a model-chosen name
// ("Налоги", "Покупки в ИКЕА", ...). Topics belong to the chat and are
// shared across sessions, like Memory.
type AdvisorTopic struct {
	ID          int
	Name        string
	Entries     []*AdvisorEntry
	NextEntryID int
}

// FindAdvisorTopic looks up a topic by ID. Returns nil if not found.
func (c *Chat) FindAdvisorTopic(id int) *AdvisorTopic {
	for _, t := range c.AdvisorTopics {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// FindEntry looks up an entry by ID inside the topic. Returns nil if not found.
func (t *AdvisorTopic) FindEntry(id int) *AdvisorEntry {
	for _, e := range t.Entries {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// FindAdvisorTopicByName looks up a topic by name, case-insensitively.
// Returns nil if not found.
func (c *Chat) FindAdvisorTopicByName(name string) *AdvisorTopic {
	name = strings.TrimSpace(name)
	for _, t := range c.AdvisorTopics {
		if strings.EqualFold(t.Name, name) {
			return t
		}
	}
	return nil
}

// AddAdvisorNote appends a note to the topic with the given name, creating
// the topic if it does not exist yet. Returns the topic and the new entry.
func (c *Chat) AddAdvisorNote(topicName, text string) (*AdvisorTopic, *AdvisorEntry) {
	topic := c.FindAdvisorTopicByName(topicName)
	if topic == nil {
		if c.NextAdvisorTopicID == 0 {
			c.NextAdvisorTopicID = 1
		}
		topic = &AdvisorTopic{
			ID:          c.NextAdvisorTopicID,
			Name:        strings.TrimSpace(topicName),
			NextEntryID: 1,
		}
		c.NextAdvisorTopicID++
		c.AdvisorTopics = append(c.AdvisorTopics, topic)
	}
	if topic.NextEntryID == 0 {
		topic.NextEntryID = 1
	}
	entry := &AdvisorEntry{
		ID:      topic.NextEntryID,
		Text:    strings.TrimSpace(text),
		Created: time.Now(),
	}
	topic.NextEntryID++
	topic.Entries = append(topic.Entries, entry)
	return topic, entry
}

// RemoveAdvisorEntry deletes one entry from a topic. A topic left without
// entries is removed as well. Returns false if topic or entry is not found.
func (c *Chat) RemoveAdvisorEntry(topicID, entryID int) bool {
	topic := c.FindAdvisorTopic(topicID)
	if topic == nil {
		return false
	}
	for i, e := range topic.Entries {
		if e.ID == entryID {
			topic.Entries = append(topic.Entries[:i], topic.Entries[i+1:]...)
			if len(topic.Entries) == 0 {
				c.RemoveAdvisorTopic(topicID)
			}
			return true
		}
	}
	return false
}

// RemoveAdvisorTopic deletes a topic with all its entries.
// Returns false if not found.
func (c *Chat) RemoveAdvisorTopic(id int) bool {
	for i, t := range c.AdvisorTopics {
		if t.ID == id {
			c.AdvisorTopics = append(c.AdvisorTopics[:i], c.AdvisorTopics[i+1:]...)
			return true
		}
	}
	return false
}

// MergeAdvisorTopics moves every entry of src into dst (renumbered) and
// removes src. Returns false when either topic is missing or src == dst.
func (c *Chat) MergeAdvisorTopics(srcID, dstID int) bool {
	if srcID == dstID {
		return false
	}
	src := c.FindAdvisorTopic(srcID)
	dst := c.FindAdvisorTopic(dstID)
	if src == nil || dst == nil {
		return false
	}
	if dst.NextEntryID == 0 {
		dst.NextEntryID = 1
	}
	for _, e := range src.Entries {
		dst.Entries = append(dst.Entries, &AdvisorEntry{
			ID:       dst.NextEntryID,
			Text:     e.Text,
			Created:  e.Created,
			RemindAt: e.RemindAt,
		})
		dst.NextEntryID++
	}
	c.RemoveAdvisorTopic(srcID)
	return true
}

// SetAdvisorReminder sets or clears (at == nil) the reminder of one entry.
// Returns false when the topic or entry is not found.
func (c *Chat) SetAdvisorReminder(topicID, entryID int, at *time.Time) bool {
	topic := c.FindAdvisorTopic(topicID)
	if topic == nil {
		return false
	}
	entry := topic.FindEntry(entryID)
	if entry == nil {
		return false
	}
	entry.RemindAt = at
	return true
}

// AdvisorReminder pairs a due entry with the topic it belongs to.
type AdvisorReminder struct {
	Topic *AdvisorTopic
	Entry *AdvisorEntry
}

// DueAdvisorReminders returns every entry whose reminder time has passed.
func (c *Chat) DueAdvisorReminders(now time.Time) []AdvisorReminder {
	var due []AdvisorReminder
	for _, t := range c.AdvisorTopics {
		for _, e := range t.Entries {
			if e.RemindAt != nil && !e.RemindAt.After(now) {
				due = append(due, AdvisorReminder{Topic: t, Entry: e})
			}
		}
	}
	return due
}
