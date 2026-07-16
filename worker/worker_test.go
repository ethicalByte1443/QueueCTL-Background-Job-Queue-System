package worker

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethicalByte1443/queuectl/db"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "queuectl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	dsn := "file:" + dbPath + "?_pragma=journal_mode%3DWAL&_pragma=busy_timeout%3D5000&_pragma=foreign_keys%3Don"
	
	dbConn, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	db.DB = dbConn

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
	if _, err := db.DB.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	t.Cleanup(func() {
		db.DB.Close()
		os.RemoveAll(tmpDir)
	})
}

func TestClaimAndRunJob_Success(t *testing.T) {
	setupTestDB(t)

	_, err := db.DB.Exec(
		"INSERT INTO jobs (id, command, state, attempts, max_retries, priority, timeout) VALUES (?, ?, 'pending', 0, 3, 0, 10)",
		"test-success", "echo hello",
	)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	ctx := context.Background()
	found := claimAndRunJob(ctx, 1)
	if !found {
		t.Errorf("expected to find and run job")
	}

	var state, output string
	err = db.DB.QueryRow("SELECT state, output FROM jobs WHERE id = ?", "test-success").Scan(&state, &output)
	if err != nil {
		t.Fatalf("failed to query job state: %v", err)
	}

	if state != "completed" {
		t.Errorf("expected state completed, got %s", state)
	}
	if output == "" {
		t.Errorf("expected output to be captured")
	}
}

func TestClaimAndRunJob_FailureAndRetry(t *testing.T) {
	setupTestDB(t)

	_, err := db.DB.Exec(
		"INSERT INTO jobs (id, command, state, attempts, max_retries, priority, timeout) VALUES (?, ?, 'pending', 0, 3, 0, 10)",
		"test-fail", "non_existent_command_12345",
	)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	ctx := context.Background()
	found := claimAndRunJob(ctx, 1)
	if !found {
		t.Errorf("expected to find and run job")
	}

	var state, errorMsg string
	var attempts int
	err = db.DB.QueryRow("SELECT state, attempts, error_msg FROM jobs WHERE id = ?", "test-fail").Scan(&state, &attempts, &errorMsg)
	if err != nil {
		t.Fatalf("failed to query job state: %v", err)
	}

	if state != "failed" {
		t.Errorf("expected state failed, got %s", state)
	}
	if attempts != 1 {
		t.Errorf("expected attempts 1, got %d", attempts)
	}
	if errorMsg == "" {
		t.Errorf("expected error message to be recorded")
	}
}

func TestClaimAndRunJob_DLQ(t *testing.T) {
	setupTestDB(t)

	_, err := db.DB.Exec(
		"INSERT INTO jobs (id, command, state, attempts, max_retries, priority, timeout) VALUES (?, ?, 'pending', 2, 3, 0, 10)",
		"test-dlq", "non_existent_command_12345",
	)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	ctx := context.Background()
	found := claimAndRunJob(ctx, 1)
	if !found {
		t.Errorf("expected to find and run job")
	}

	var state string
	var attempts int
	err = db.DB.QueryRow("SELECT state, attempts FROM jobs WHERE id = ?", "test-dlq").Scan(&state, &attempts)
	if err != nil {
		t.Fatalf("failed to query job state: %v", err)
	}

	if state != "dead" {
		t.Errorf("expected state dead (DLQ), got %s", state)
	}
	if attempts != 3 {
		t.Errorf("expected attempts 3, got %d", attempts)
	}
}

func TestClaimAndRunJob_PriorityOrder(t *testing.T) {
	setupTestDB(t)

	_, _ = db.DB.Exec(
		"INSERT INTO jobs (id, command, state, priority) VALUES (?, ?, 'pending', 0)",
		"low-prio", "echo low",
	)
	_, _ = db.DB.Exec(
		"INSERT INTO jobs (id, command, state, priority) VALUES (?, ?, 'pending', 10)",
		"high-prio", "echo high",
	)

	ctx := context.Background()
	found := claimAndRunJob(ctx, 1)
	if !found {
		t.Fatalf("expected to run high prio job first")
	}

	var highState, lowState string
	_ = db.DB.QueryRow("SELECT state FROM jobs WHERE id = ?", "high-prio").Scan(&highState)
	_ = db.DB.QueryRow("SELECT state FROM jobs WHERE id = ?", "low-prio").Scan(&lowState)

	if highState != "completed" {
		t.Errorf("expected high-prio completed, got %s", highState)
	}
	if lowState != "pending" {
		t.Errorf("expected low-prio still pending, got %s", lowState)
	}
}
