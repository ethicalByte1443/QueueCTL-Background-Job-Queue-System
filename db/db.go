package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".queuectl")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("could not create data directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "queuectl.db")
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode%%3DWAL&_pragma=busy_timeout%%3D5000&_pragma=foreign_keys%%3Don", dbPath)

	DB, err = sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}

	if err := createSchema(); err != nil {
		return fmt.Errorf("could not create schema: %w", err)
	}

	fmt.Printf("[DB] Initialized at: %s\n", dbPath)
	return nil
}

func createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id          TEXT PRIMARY KEY,
		command     TEXT NOT NULL,
		state       TEXT NOT NULL DEFAULT 'pending',
		attempts    INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		output      TEXT DEFAULT '',
		error_msg   TEXT DEFAULT '',
		priority    INTEGER NOT NULL DEFAULT 0,
		run_at      TEXT DEFAULT NULL,
		timeout     INTEGER NOT NULL DEFAULT 600,
		duration_ms INTEGER DEFAULT 0,
		created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);

	CREATE TABLE IF NOT EXISTS config (
		key   TEXT PRIMARY KEY,
		value INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS worker_processes (
		pid          INTEGER PRIMARY KEY,
		worker_count INTEGER NOT NULL DEFAULT 1,
		status       TEXT NOT NULL,
		started_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	`
	_, err := DB.Exec(schema)
	return err
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
