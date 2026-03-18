package storage

import (
	"time"
)

// Session defaults
const (
	DefaultSessionTopic       = "default"
	DefaultSessionModel       = "basic"
	DefaultSessionTemperature = 0.8
	DefaultSessionID          = 1
	DefaultNextSessionID      = 2
)

type Storage interface {
	Get(chatID int64) (*Chat, bool)
	Set(chatID int64, chat *Chat) error
	MarkDirty(chatID int64)
	Save() bool
}

// Session represents an independent conversation thread inside a Telegram chat.
type Session struct {
	ID           int
	Topic        string
	History      []*ConversationEntry
	SystemPrompt string
	Model        string
	Temperature  float32
}

type Chat struct {
	ChatID           int64
	Settings         ChatSettings
	Sessions         []*Session `json:",omitempty"`
	ActiveSessionID  int        `json:",omitempty"`
	NextSessionID    int        `json:",omitempty"`
	ImageGenNextTime time.Time
	Title            string

	// Legacy fields — used only for migration from old format.
	History []*ConversationEntry `json:",omitempty"`
}

type ChatSettings struct {
	MaxMessages     int
	UseMarkdown     bool
	SummarizePrompt string
	Token           string

	// Legacy fields — kept for JSON backward compat during migration.
	Temperature  float32 `json:",omitempty"`
	Model        string  `json:",omitempty"`
	SystemPrompt string  `json:",omitempty"`
}

type ConversationEntry struct {
	Prompt   Message
	Response Message
}

type Message struct {
	Role    string
	Content string
}

// ActiveSession returns the currently selected session.
// Always returns a valid session: falls back to the first one,
// or auto-creates a "default" session if none exist.
func (c *Chat) ActiveSession() *Session {
	for _, s := range c.Sessions {
		if s.ID == c.ActiveSessionID {
			return s
		}
	}
	if len(c.Sessions) > 0 {
		c.ActiveSessionID = c.Sessions[0].ID
		return c.Sessions[0]
	}

	// Should never happen, but guarantee non-nil.
	s := &Session{
		ID:          DefaultSessionID,
		Topic:       DefaultSessionTopic,
		History:     make([]*ConversationEntry, 0),
		Model:       DefaultSessionModel,
		Temperature: DefaultSessionTemperature,
	}
	c.Sessions = []*Session{s}
	c.ActiveSessionID = DefaultSessionID
	c.NextSessionID = DefaultNextSessionID
	return s
}

// FindSession looks up a session by ID. Returns nil if not found.
func (c *Chat) FindSession(id int) *Session {
	for _, s := range c.Sessions {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// RemoveSession deletes a session by ID. Cannot remove the last session.
// Returns false if not found or if it's the only session.
func (c *Chat) RemoveSession(id int) bool {
	if len(c.Sessions) <= 1 {
		return false
	}
	for i, s := range c.Sessions {
		if s.ID == id {
			c.Sessions = append(c.Sessions[:i], c.Sessions[i+1:]...)
			if c.ActiveSessionID == id {
				c.ActiveSessionID = c.Sessions[0].ID
			}
			return true
		}
	}
	return false
}

// AddSession creates a new session with the given topic, inheriting
// Model and Temperature from the currently active session.
func (c *Chat) AddSession(topic string) *Session {
	active := c.ActiveSession()
	model := DefaultSessionModel
	temp := float32(DefaultSessionTemperature)
	if active != nil {
		model = active.Model
		temp = active.Temperature
	}

	s := &Session{
		ID:          c.NextSessionID,
		Topic:       topic,
		History:     make([]*ConversationEntry, 0),
		Model:       model,
		Temperature: temp,
	}
	c.NextSessionID++
	c.Sessions = append(c.Sessions, s)
	return s
}

// Migrate converts a legacy Chat (pre-sessions) into the new format.
// Safe to call multiple times — it's a no-op if Sessions already exist.
func (c *Chat) Migrate() {
	if len(c.Sessions) > 0 {
		return
	}

	c.NextSessionID = DefaultNextSessionID
	c.ActiveSessionID = DefaultSessionID
	c.Sessions = []*Session{{
		ID:           DefaultSessionID,
		Topic:        DefaultSessionTopic,
		History:      c.History,
		SystemPrompt: c.Settings.SystemPrompt,
		Model:        c.Settings.Model,
		Temperature:  c.Settings.Temperature,
	}}

	if c.Sessions[0].History == nil {
		c.Sessions[0].History = make([]*ConversationEntry, 0)
	}
	if c.Sessions[0].Model == "" {
		c.Sessions[0].Model = DefaultSessionModel
	}
	if c.Sessions[0].Temperature == 0 {
		c.Sessions[0].Temperature = DefaultSessionTemperature
	}

	// Clear legacy fields
	c.History = nil
	c.Settings.SystemPrompt = ""
	c.Settings.Model = ""
	c.Settings.Temperature = 0
}
