package storage

import (
	"GPTBot/domain/chat"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// SQLiteStorage stores chats in a SQLite database.
// Writes happen immediately (no dirty tracking needed).
type SQLiteStorage struct {
	mu    sync.RWMutex
	db    *sql.DB
	cache map[int64]*chat.Chat
}

func NewSQLiteStorage(dsn string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite WAL: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite migrate: %w", err)
	}

	return &SQLiteStorage{
		db:    db,
		cache: make(map[int64]*chat.Chat),
	}, nil
}

// migrate creates tables if they don't exist.
func migrate(db *sql.DB) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS chats (
		chat_id  INTEGER PRIMARY KEY,
		payload  TEXT NOT NULL
	);`
	_, err := db.Exec(ddl)
	return err
}

func (s *SQLiteStorage) Get(chatID int64) (*chat.Chat, bool) {
	s.mu.RLock()
	if c, ok := s.cache[chatID]; ok {
		s.mu.RUnlock()
		return c, true
	}
	s.mu.RUnlock()

	// Load from DB.
	row := s.db.QueryRow(`SELECT payload FROM chats WHERE chat_id = ?`, chatID)
	var payload string
	if err := row.Scan(&payload); err != nil {
		return nil, false
	}

	var c chat.Chat
	if err := json.Unmarshal([]byte(payload), &c); err != nil {
		return nil, false
	}

	// Back-compat: old data without sessions.
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

	s.mu.Lock()
	s.cache[chatID] = &c
	s.mu.Unlock()

	return &c, true
}

func (s *SQLiteStorage) Set(chatID int64, c *chat.Chat) error {
	s.mu.Lock()
	s.cache[chatID] = c
	s.mu.Unlock()

	return s.persist(chatID, c)
}

func (s *SQLiteStorage) MarkDirty(chatID int64) {
	s.mu.RLock()
	c, ok := s.cache[chatID]
	s.mu.RUnlock()

	if ok {
		_ = s.persist(chatID, c)
	}
}

func (s *SQLiteStorage) Save() bool {
	s.mu.RLock()
	snapshot := make(map[int64]*chat.Chat, len(s.cache))
	for k, v := range s.cache {
		snapshot[k] = v
	}
	s.mu.RUnlock()

	for chatID, c := range snapshot {
		if err := s.persist(chatID, c); err != nil {
			return false
		}
	}
	return true
}

// persist writes a single chat to the database.
func (s *SQLiteStorage) persist(chatID int64, c *chat.Chat) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO chats (chat_id, payload) VALUES (?, ?)
		 ON CONFLICT(chat_id) DO UPDATE SET payload = excluded.payload`,
		chatID, string(data),
	)
	return err
}

// Close closes the underlying database connection.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
