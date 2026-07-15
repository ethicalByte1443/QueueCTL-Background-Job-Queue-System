/*
=============================================================================
🎓 LEARNING NOTE — db/db.go (Database Persistence Layer)
=============================================================================

WHAT THIS FILE DOES:
  This file sets up a connection to an SQLite database and creates the "jobs"
  table if it doesn't already exist. Every other part of the app (enqueue,
  worker, status, etc.) will import this package to read/write jobs.

KEY GO CONCEPTS USED HERE:

  1. PACKAGE DECLARATION — `package db`
     Every .go file starts with a "package" line. It groups related code.
     Files in the same folder must share the same package name.
     Other files import this as: import "github.com/ethicalByte1443/queuectl/db"

  2. IMPORTS
     Go has a built-in module system. You list everything you need at the top.
     Standard library packages (like "fmt") have short names.
     External packages (like the sqlite driver) use full URL paths.

  3. FUNCTIONS
     func FunctionName(param Type) ReturnType { ... }
     Functions starting with an UPPERCASE letter (like "InitDB") are PUBLIC —
     they can be called from other packages.
     Functions starting with a lowercase letter are PRIVATE to this package.

  4. ERROR HANDLING
     Go does NOT have try/catch. Instead, functions return an "error" as a
     second value. You check it immediately:
         db, err := sql.Open(...)
         if err != nil { return err }
     This pattern is EVERYWHERE in Go. It forces you to handle every error.

  5. *sql.DB (POINTER)
     The * means "pointer to". A pointer is just an address in memory pointing
     to the real object. We pass pointers around so every part of the app uses
     the SAME database connection, not a copy of it.

  6. DEFER
     `defer db.Close()` means: "run this line LATER, when the surrounding
     function exits." It's Go's way of ensuring cleanup always happens.
     We DON'T use defer for the DB connection here because we want the
     connection to stay open for the entire app lifetime.

=============================================================================
*/

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// This blank import "_" registers the SQLite driver with Go's database/sql
	// package. We don't call it directly — it runs an init() function behind
	// the scenes that says "hey, I can handle sqlite databases."
	// We use modernc.org/sqlite because it's written in pure Go — no C
	// compiler needed. Perfect for Windows!
	_ "modernc.org/sqlite"
)

// DB is the global database connection. Other packages will use db.DB to
// run queries. We keep it as a package-level variable so it's shared.
var DB *sql.DB

// InitDB opens (or creates) the SQLite database file and creates the jobs
// table if it doesn't exist yet.
//
// It returns an error if anything goes wrong (e.g., file permission issues).
func InitDB() error {

	// --- Step 1: Determine where to store the database file ---
	// We store it in the user's home directory under .queuectl/queuectl.db
	// This way, the DB persists across sessions and is not tied to the
	// current working directory.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".qcli")

	// os.MkdirAll creates the directory (and any parents) if they don't exist.
	// 0755 is a Unix permission code: owner can read/write/execute, others can
	// read/execute. On Windows this is mostly ignored, but it's good practice.
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("could not create data directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "qcli.db")

	// --- Step 2: Open the database connection ---
	// sql.Open doesn't actually connect — it just prepares the connection.
	// The "sqlite3" string tells Go to use the driver we imported above.
	// The options after "?" configure SQLite:
	//   _journal_mode=WAL  → Write-Ahead Logging for better concurrent reads.
	//   _busy_timeout=5000 → Wait up to 5 seconds if the DB is locked by
	//                        another worker, instead of failing immediately.
	//   _foreign_keys=on   → Enforce foreign key constraints.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode%%3DWAL&_pragma=busy_timeout%%3D5000&_pragma=foreign_keys%%3Don", dbPath)

	DB, err = sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}

	// --- Step 3: Verify the connection actually works ---
	// Ping sends a test query to make sure the file is readable/writable.
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}

	// --- Step 4: Create the jobs table if it doesn't exist ---
	if err := createSchema(); err != nil {
		return fmt.Errorf("could not create schema: %w", err)
	}

	fmt.Printf("[DB] Initialized at: %s\n", dbPath)
	return nil
}

// createSchema runs the SQL to create the "jobs" table.
// This is called every time the app starts, but "IF NOT EXISTS" ensures
// it only creates the table if it's missing — it won't delete existing data.
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
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- This index speeds up the worker's query to find pending/failed jobs.
	-- Without it, SQLite would scan every row in the table every time.
	CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);

	-- Config table stores key-value pairs for runtime settings.
	-- Examples: max_retries=3, backoff_base=2
	CREATE TABLE IF NOT EXISTS config (
		key   TEXT PRIMARY KEY,
		value INTEGER NOT NULL
	);
	`

	// Exec runs SQL that doesn't return rows (CREATE, INSERT, UPDATE, DELETE).
	_, err := DB.Exec(schema)
	return err
}

// Close cleanly shuts down the database connection.
// This should be called when the application exits (typically via defer in main).
func Close() {
	if DB != nil {
		DB.Close()
	}
}
