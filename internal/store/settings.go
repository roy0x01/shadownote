package store

import (
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) ConfigBlob() (string, bool, error) {
	var data string
	err := s.db.QueryRow(`SELECT data FROM settings WHERE id = 1`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read settings: %w", err)
	}
	return data, true, nil
}

func (s *Store) PutConfigBlob(data string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (id, data, updated) VALUES (1, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET data = excluded.data, updated = datetime('now')`, data)
	if err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}
