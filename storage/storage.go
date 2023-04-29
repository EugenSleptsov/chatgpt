package storage

import (
	"time"
)

type Storage interface {
	Get(chatID int64) (*Chat, bool)
	Set(chatID int64, chat *Chat) error
	Save() bool
}

type Chat struct {
	ChatID           int64
	Settings         ChatSettings
	History          []*ConversationEntry
	ImageGenNextTime time.Time
}

type ChatSettings struct {
	Temperature float32
	Model       string
	MaxMessages int
	UseMarkdown bool
}

type ConversationEntry struct {
	Prompt   Message
	Response Message
}

type Message struct {
	Role    string
	Content string
}
