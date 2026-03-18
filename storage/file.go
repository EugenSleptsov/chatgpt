package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStorage struct {
	mu      sync.RWMutex
	dirPath string
	chats   map[int64]*Chat
	dirty   map[int64]bool
}

// constructor

func NewFileStorage(dirPath string) (*FileStorage, error) {
	storage := &FileStorage{
		dirPath: dirPath,
		chats:   make(map[int64]*Chat),
		dirty:   make(map[int64]bool),
	}

	// check that dirPath exists
	if !storage.dirExists() {
		if err := os.Mkdir(storage.dirPath, 0755); err != nil {
			return nil, err
		}
	}

	return storage, nil
}

// implementation

func (s *FileStorage) Get(chatID int64) (*Chat, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chat, ok := s.chats[chatID]
	if !ok {
		var err error
		chat, err = s.loadChatFromFile(chatID)
		if err != nil {
			return nil, false
		}
		s.chats[chatID] = chat
		ok = true
	}

	return chat, ok
}

func (s *FileStorage) Set(chatID int64, chat *Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chats[chatID] = chat
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
		if chat, ok := s.chats[chatID]; ok {
			if err := s.saveChatToFile(chatID, chat); err != nil {
				success = false
			}
		}
	}

	s.dirty = make(map[int64]bool)
	return success
}

// helpers

func (s *FileStorage) dirExists() bool {
	if _, err := os.Stat(s.dirPath); os.IsNotExist(err) {
		return false
	}
	return true
}
func (s *FileStorage) chatFilePath(chatID int64) string {
	return filepath.Join(s.dirPath, fmt.Sprintf("%d.chat", chatID))
}

func (s *FileStorage) loadChatFromFile(chatID int64) (*Chat, error) {
	filePath := s.chatFilePath(chatID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var chat Chat
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, err
	}

	chat.Migrate()
	return &chat, nil
}

func (s *FileStorage) saveChatToFile(chatID int64, chat *Chat) error {
	filePath := s.chatFilePath(chatID)
	data, err := json.Marshal(chat)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
