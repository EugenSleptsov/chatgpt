package storage

import (
	"GPTBot/domain/chat"
	"GPTBot/infrastructure/util"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStorage struct {
	mu      sync.RWMutex
	dirPath string
	chats   map[int64]*chat.Chat
	dirty   map[int64]bool
}

// constructor

func NewFileStorage(dirPath string) (*FileStorage, error) {
	storage := &FileStorage{
		dirPath: dirPath,
		chats:   make(map[int64]*chat.Chat),
		dirty:   make(map[int64]bool),
	}

	// check that dirPath exists
	if !util.IsDirExists(storage.dirPath) {
		if err := os.Mkdir(storage.dirPath, 0755); err != nil {
			return nil, err
		}
	}

	return storage, nil
}

// implementation

func (s *FileStorage) Get(chatID int64) (*chat.Chat, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.chats[chatID]
	if !ok {
		var err error
		c, err = s.loadChatFromFile(chatID)
		if err != nil {
			return nil, false
		}
		s.chats[chatID] = c
		ok = true
	}
	return c, ok
}

func (s *FileStorage) Set(chatID int64, c *chat.Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chats[chatID] = c
	s.dirty[chatID] = true
	return nil
}

func (s *FileStorage) MarkDirty(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dirty[chatID] = true
}

func (s *FileStorage) Save() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.dirty) == 0 {
		return true
	}

	success := true
	for chatID := range s.dirty {
		if c, ok := s.chats[chatID]; ok {
			if err := s.saveChatToFile(chatID, c); err != nil {
				success = false
			}
		}
	}
	s.dirty = make(map[int64]bool)
	return success
}

// helpers

func (s *FileStorage) chatFilePath(chatID int64) string {
	return filepath.Join(s.dirPath, fmt.Sprintf("%d.chat", chatID))
}

func (s *FileStorage) loadChatFromFile(chatID int64) (*chat.Chat, error) {
	filePath := s.chatFilePath(chatID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var c chat.Chat
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	// Unsupported old format (no sessions) — start fresh.
	if len(c.Sessions) == 0 {
		c.Sessions = []*chat.Session{{
			ID:      chat.DefaultSessionID,
			Topic:   chat.DefaultSessionTopic,
			History: make([]*chat.ConversationEntry, 0),
			Model:   chat.DefaultSessionModel,
		}}
		c.ActiveSessionID = chat.DefaultSessionID
		c.NextSessionID = chat.DefaultNextSessionID
	}

	return &c, nil
}

func (s *FileStorage) saveChatToFile(chatID int64, c *chat.Chat) error {
	filePath := s.chatFilePath(chatID)
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// All loads every .chat file from the directory and returns a map of chatID → *chat.Chat.
// Used by the migrator.
func (s *FileStorage) All() (map[int64]*chat.Chat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dirPath)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]*chat.Chat)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".chat" {
			continue
		}
		var chatID int64
		if _, err := fmt.Sscanf(name, "%d.chat", &chatID); err != nil {
			continue
		}
		c, err := s.loadChatFromFile(chatID)
		if err != nil {
			continue
		}
		result[chatID] = c
	}
	return result, nil
}
