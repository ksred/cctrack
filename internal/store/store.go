package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// ResetParsedData clears the sessions, requests, and file_offsets tables so a
// subsequent ParseAll re-ingests every JSONL log from the beginning. Used by
// `cctrack reset` to recover from incomplete state — e.g. when an older binary
// populated sessions but the requests table didn't exist yet, leaving per-hour
// queries blind to historical data.
func (s *Store) ResetParsedData() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, q := range []string{
		"DELETE FROM requests",
		"DELETE FROM sessions",
		"DELETE FROM file_offsets",
	} {
		if _, err := tx.Exec(q); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id            TEXT PRIMARY KEY,
			project       TEXT NOT NULL DEFAULT '',
			slug          TEXT NOT NULL DEFAULT '',
			model         TEXT NOT NULL DEFAULT '',
			started_at    TEXT NOT NULL DEFAULT '',
			last_activity TEXT NOT NULL DEFAULT '',
			total_input   INTEGER NOT NULL DEFAULT 0,
			total_output  INTEGER NOT NULL DEFAULT 0,
			total_cache_read     INTEGER NOT NULL DEFAULT 0,
			total_cache_write_5m INTEGER NOT NULL DEFAULT 0,
			total_cache_write_1h INTEGER NOT NULL DEFAULT 0,
			total_cost    REAL NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS file_offsets (
			path       TEXT PRIMARY KEY,
			offset     INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS requests (
			request_id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			timestamp  TEXT NOT NULL DEFAULT '',
			model      TEXT NOT NULL DEFAULT '',
			input_tokens         INTEGER NOT NULL DEFAULT 0,
			output_tokens        INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens    INTEGER NOT NULL DEFAULT 0,
			cache_write_5m_tokens INTEGER NOT NULL DEFAULT 0,
			cache_write_1h_tokens INTEGER NOT NULL DEFAULT 0,
			cost       REAL NOT NULL DEFAULT 0,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);

		CREATE TABLE IF NOT EXISTS window_anchors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			synced_at  TEXT NOT NULL,
			window_type TEXT NOT NULL,
			time_left_minutes INTEGER NOT NULL,
			anthropic_pct REAL,
			observed_cost REAL NOT NULL DEFAULT 0,
			inferred_cap REAL
		);
		CREATE INDEX IF NOT EXISTS idx_window_anchors_type_synced
			ON window_anchors(window_type, synced_at DESC);

		CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity);
		CREATE INDEX IF NOT EXISTS idx_sessions_total_cost ON sessions(total_cost);
		CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at);
		CREATE INDEX IF NOT EXISTS idx_requests_session_id ON requests(session_id);
		CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
	`)
	return err
}
