package storage

import (
	"GPTBot/domain/chat"
	"sync"
)

// MemoryStorage is a purely in-memory Storage implementation.
// Data is lost when the process exits. Useful for tests and ephemeral bots.
type MemoryStorage struct {
	mu    sync.RWMutex
	chats map[int64]*chat.Chat
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		chats: make(map[int64]*chat.Chat),
	}
}

func (m *MemoryStorage) Get(chatID int64) (*chat.Chat, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.chats[chatID]
	return c, ok
}

func (m *MemoryStorage) Set(chatID int64, c *chat.Chat) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.chats[chatID] = c
	return nil
}

func (m *MemoryStorage) MarkDirty(_ int64) {
	// no-op: everything is in memory
}

func (m *MemoryStorage) Save() bool {
	// no-op: nothing to persist
	return true
}

// All returns a snapshot of every chat in memory. Used by the migrator.
func (m *MemoryStorage) All() map[int64]*chat.Chat {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[int64]*chat.Chat, len(m.chats))
	for k, v := range m.chats {
		out[k] = v
	}
	return out
}
