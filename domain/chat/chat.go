// Package chat contains the core domain entities for chat management.
// These types are transport-agnostic and must not depend on any
// infrastructure or integration packages.
package chat

import "time"

// Session defaults
const (
	DefaultSessionTopic  = "default"
	DefaultSessionModel  = "basic"
	DefaultSessionID     = 1
	DefaultNextSessionID = 2
)

// Storage is the repository interface for persisting Chat objects.
// Implementations live in infrastructure/storage.
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
}

// Chat is the aggregate root for a Telegram chat.
type Chat struct {
	ChatID           int64
	Settings         ChatSettings
	Sessions         []*Session `json:",omitempty"`
	ActiveSessionID  int        `json:",omitempty"`
	NextSessionID    int        `json:",omitempty"`
	ImageGenNextTime time.Time
	Title            string
	Memory           []string

	// Persisted across restarts so the admin can monitor per-chat spend.
	TotalCostUSD      float64   // accumulated USD cost
	TotalInputTokens  int       // accumulated input tokens
	TotalOutputTokens int       // accumulated output tokens
	TotalRequests     int       // number of GPT API calls
	CostResetTime     time.Time // when the daily cost counter was last reset
}

// ChatSettings holds per-chat configuration.
type ChatSettings struct {
	MaxMessages      int
	UseMarkdown      bool
	SummarizePrompt  string
	GroupAutoReply   bool    // bot proactively joins group conversations
	AutoReplyPersona string  // configurable role/persona for the auto-reply decision prompt (empty = use global default)
	CostLimitUSD     float64 // daily cost limit in USD, 0 = unlimited
}

// ConversationEntry stores one prompt/response pair in the session history.
type ConversationEntry struct {
	Prompt   Message
	Response Message
}

// Message is a single chat message with a role and text content.
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
		ID:      DefaultSessionID,
		Topic:   DefaultSessionTopic,
		History: make([]*ConversationEntry, 0),
		Model:   DefaultSessionModel,
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
// Model from the currently active session.
func (c *Chat) AddSession(topic string) *Session {
	active := c.ActiveSession()
	model := DefaultSessionModel
	if active != nil {
		model = active.Model
	}

	s := &Session{
		ID:      c.NextSessionID,
		Topic:   topic,
		History: make([]*ConversationEntry, 0),
		Model:   model,
	}
	c.NextSessionID++
	c.Sessions = append(c.Sessions, s)
	return s
}

// AccumulateCost adds usage from a single request to the running totals.
// Resets counters if a new calendar day has started (daily budget cycle).
func (c *Chat) AccumulateCost(cost float64, inputTokens, outputTokens int) {
	now := time.Now()
	if !sameDay(c.CostResetTime, now) {
		c.TotalCostUSD = 0
		c.TotalInputTokens = 0
		c.TotalOutputTokens = 0
		c.TotalRequests = 0
		c.CostResetTime = now
	}
	c.TotalCostUSD += cost
	c.TotalInputTokens += inputTokens
	c.TotalOutputTokens += outputTokens
	c.TotalRequests++
}

// CostLimitExceeded returns true if the chat has exceeded its daily cost limit.
// A zero limit means unlimited.
func (c *Chat) CostLimitExceeded(limitUSD float64) bool {
	if limitUSD <= 0 {
		return false
	}
	// Reset if new day
	if !sameDay(c.CostResetTime, time.Now()) {
		return false
	}
	return c.TotalCostUSD >= limitUSD
}

func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
