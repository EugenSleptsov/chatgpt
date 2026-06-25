package storage

import (
	"GPTBot/domain/chat"
	"fmt"
)

// NewStorage creates a Storage implementation based on the given type string.
//
//	"file"   — JSON files in dataDir  (default)
//	"memory" — ephemeral in-memory store
func NewStorage(storageType, dataDir string) (chat.Storage, error) {
	switch storageType {
	case "file", "":
		return NewFileStorage(dataDir)
	case "memory":
		return NewMemoryStorage(), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %q (supported: file, memory)", storageType)
	}
}
