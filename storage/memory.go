package storage

type MemoryStorage struct {
	chats map[int64]*Chat
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		chats: make(map[int64]*Chat),
	}
}

func (m *MemoryStorage) Get(chatID int64) (*Chat, bool) {
	chat, ok := m.chats[chatID]
	return chat, ok
}

func (m *MemoryStorage) Set(chatID int64, chat *Chat) error {
	m.chats[chatID] = chat
	return nil
}
