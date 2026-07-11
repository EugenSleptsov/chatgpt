package storage

import (
	"GPTBot/domain/chat"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// NewArchive creates an Archive implementation matching the storage type:
//
//	"file"   — append-only JSONL files in dataDir/archive (default)
//	"memory" — ephemeral in-memory archive
func NewArchive(storageType, dataDir string) (chat.Archive, error) {
	switch storageType {
	case "file", "":
		return NewFileArchive(dataDir)
	case "memory":
		return NewMemoryArchive(), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %q (supported: file, memory)", storageType)
	}
}

// FileArchive stores each chat's raw transcript as an append-only JSONL file
// <dataDir>/archive/<chatID>.jsonl. The zero-based line index is the archive
// address; MemoryNode ranges refer to these indices, so lines are never
// reordered or removed (future hard-delete will tombstone in place).
type FileArchive struct {
	mu    sync.Mutex
	dir   string
	count map[int64]int // cached line count per chat (next append index)
}

func NewFileArchive(dataDir string) (*FileArchive, error) {
	dir := filepath.Join(dataDir, "archive")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &FileArchive{dir: dir, count: make(map[int64]int)}, nil
}

func (a *FileArchive) path(chatID int64) string {
	return filepath.Join(a.dir, fmt.Sprintf("%d.jsonl", chatID))
}

// lineCount returns the number of lines in the chat's archive, counting the
// file once per process lifetime and serving from cache afterwards.
// Caller must hold a.mu.
func (a *FileArchive) lineCount(chatID int64) (int, error) {
	if n, ok := a.count[chatID]; ok {
		return n, nil
	}
	f, err := os.Open(a.path(chatID))
	if err != nil {
		if os.IsNotExist(err) {
			a.count[chatID] = 0
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()
	n := 0
	sc := newArchiveScanner(f)
	for sc.Scan() {
		n++
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	a.count[chatID] = n
	return n, nil
}

func (a *FileArchive) Append(chatID int64, msgs []chat.ArchivedMessage) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	from, err := a.lineCount(chatID)
	if err != nil {
		return 0, err
	}
	f, err := os.OpenFile(a.path(chatID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Any failure below may leave a partial append on disk (bufio can flush
	// earlier lines before the error), so the cached count is dropped and
	// recounted lazily on the next call.
	w := bufio.NewWriter(f)
	for _, m := range msgs {
		line, err := json.Marshal(m)
		if err != nil {
			delete(a.count, chatID)
			return 0, err
		}
		if _, err := w.Write(append(line, '\n')); err != nil {
			delete(a.count, chatID)
			return 0, err
		}
	}
	if err := w.Flush(); err != nil {
		delete(a.count, chatID)
		return 0, err
	}
	a.count[chatID] = from + len(msgs)
	return from, nil
}

func (a *FileArchive) ReadRange(chatID int64, from, to int) ([]chat.ArchivedMessage, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	f, err := os.Open(a.path(chatID))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []chat.ArchivedMessage
	sc := newArchiveScanner(f)
	for i := 0; sc.Scan() && i < to; i++ {
		if i < from {
			continue
		}
		var m chat.ArchivedMessage
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			return nil, fmt.Errorf("archive line %d corrupt: %w", i, err)
		}
		out = append(out, m)
	}
	return out, sc.Err()
}

// Count returns the total number of lines stored for the chat.
func (a *FileArchive) Count(chatID int64) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lineCount(chatID)
}

// DeleteRange is the reserved privacy hook ("hard forget"). Tombstoning lines
// in place is planned but not implemented yet.
func (a *FileArchive) DeleteRange(chatID int64, from, to int) error {
	return fmt.Errorf("archive hard-delete is not implemented yet")
}

// newArchiveScanner returns a line scanner with a buffer large enough for
// long transcript lines (Telegram messages can reach tens of KB as JSON).
func newArchiveScanner(f *os.File) *bufio.Scanner {
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	return sc
}

// MemoryArchive is a purely in-memory Archive implementation for tests and
// ephemeral bots.
type MemoryArchive struct {
	mu   sync.Mutex
	msgs map[int64][]chat.ArchivedMessage
}

func NewMemoryArchive() *MemoryArchive {
	return &MemoryArchive{msgs: make(map[int64][]chat.ArchivedMessage)}
}

func (a *MemoryArchive) Append(chatID int64, msgs []chat.ArchivedMessage) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	from := len(a.msgs[chatID])
	a.msgs[chatID] = append(a.msgs[chatID], msgs...)
	return from, nil
}

func (a *MemoryArchive) ReadRange(chatID int64, from, to int) ([]chat.ArchivedMessage, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	all := a.msgs[chatID]
	if from < 0 || from > to || to > len(all) {
		return nil, fmt.Errorf("range [%d, %d) out of bounds (have %d lines)", from, to, len(all))
	}
	out := make([]chat.ArchivedMessage, to-from)
	copy(out, all[from:to])
	return out, nil
}

func (a *MemoryArchive) Count(chatID int64) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.msgs[chatID]), nil
}

func (a *MemoryArchive) DeleteRange(chatID int64, from, to int) error {
	return fmt.Errorf("archive hard-delete is not implemented yet")
}
