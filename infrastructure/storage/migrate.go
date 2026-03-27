package storage

import "fmt"

// MigrateFileToSQLite reads every chat from the file-based storage directory
// and inserts it into a SQLite database at the given DSN.
//
// Usage:
//
//	err := storage.MigrateFileToSQLite("data", "data/chats.db")
func MigrateFileToSQLite(dataDir, dsn string) error {
	src, err := NewFileStorage(dataDir)
	if err != nil {
		return fmt.Errorf("open file storage: %w", err)
	}

	allChats, err := src.All()
	if err != nil {
		return fmt.Errorf("read file storage: %w", err)
	}

	if len(allChats) == 0 {
		return fmt.Errorf("no chat files found in %s", dataDir)
	}

	dst, err := NewSQLiteStorage(dsn)
	if err != nil {
		return fmt.Errorf("open sqlite storage: %w", err)
	}
	defer dst.Close()

	var migrated int
	for chatID, chat := range allChats {
		if err := dst.Set(chatID, chat); err != nil {
			return fmt.Errorf("writing chat %d: %w", chatID, err)
		}
		migrated++
	}

	fmt.Printf("Migration complete: %d chat(s) moved from %s → %s\n", migrated, dataDir, dsn)
	return nil
}
