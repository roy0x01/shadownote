package store

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db   *sql.DB
	fts5 bool
}

// SQLite stays single-writer; WAL improves concurrent reads.
func Open(path string) (*Store, error) {
	dsn := path + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			slug      TEXT PRIMARY KEY,
			title     TEXT NOT NULL,
			type      TEXT NOT NULL,
			status    TEXT NOT NULL,
			tags      TEXT NOT NULL DEFAULT '',
			path      TEXT NOT NULL,
			created   TEXT,
			modified  TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS settings (
			id      INTEGER PRIMARY KEY CHECK (id = 1),
			data    TEXT NOT NULL,
			updated TEXT DEFAULT (datetime('now'))
		)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	if err := s.migrateSearch(); err != nil {
		return err
	}
	return nil
}

func (s *Store) migrateSearch() error {
	var existingSQL string
	err := s.db.QueryRow(`SELECT COALESCE(sql, '') FROM sqlite_master WHERE name = 'search'`).Scan(&existingSQL)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("inspect search table: %w", err)
	}
	if err == nil {
		s.fts5 = strings.Contains(strings.ToLower(existingSQL), "using fts5")
		return nil
	}

	_, err = s.db.Exec(`CREATE VIRTUAL TABLE search USING fts5(
		slug UNINDEXED,
		title,
		body,
		tags
	)`)
	if err == nil {
		s.fts5 = true
		return nil
	}
	if !strings.Contains(err.Error(), "no such module: fts5") {
		return fmt.Errorf("migrate search: %w", err)
	}

	// fts5 is optional; fall back to LIKE so plain go install still works.
	if _, err := s.db.Exec(`CREATE TABLE search (
		slug  TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		body  TEXT NOT NULL,
		tags  TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return fmt.Errorf("migrate fallback search: %w", err)
	}
	s.fts5 = false
	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	return s.db.Close()
}
