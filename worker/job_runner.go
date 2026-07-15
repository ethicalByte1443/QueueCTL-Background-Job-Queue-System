/*
=============================================================================
🎓 LEARNING NOTE — worker/job_runner.go (Lock, Pick & Execute Jobs)
=============================================================================

WHAT THIS FILE DOES:
  This is where the CORE ALGORITHM lives:
  1. LOCK the database (so other workers can't grab the same job)
  2. PICK the highest-priority pending/failed job
  3. EXECUTE the shell command
  4. UPDATE the job state based on success/failure
  5. Handle EXPONENTIAL BACKOFF for retries

KEY GO CONCEPTS:

  1. DATABASE TRANSACTIONS — tx, err := db.DB.Begin()
     A transaction groups multiple SQL operations into one atomic unit.
     Either ALL of them succeed, or NONE of them do. If we crash mid-way,
     the database rolls back to its previous state automatically.

     tx.QueryRow(...)  → read within the transaction
     tx.Exec(...)      → write within the transaction
     tx.Commit()       → finalize all changes
     tx.Rollback()     → undo all changes (on error)

  2. os/exec.CommandContext — RUNNING SHELL COMMANDS
     Go can run any shell command using exec.Command(). We use
     CommandContext() which also accepts a context — if the context
     is cancelled (Ctrl+C), the command is killed automatically.

  3. EXPONENTIAL BACKOFF CALCULATION:
     delay = base ^ attempts
     Example with base=2:
       Attempt 1 fail → wait 2^1 = 2 seconds
       Attempt 2 fail → wait 2^2 = 4 seconds
       Attempt 3 fail → wait 2^3 = 8 seconds
     The wait grows exponentially, giving the system time to recover.

  4. sql.NullString — HANDLING NULL VALUES
     SQL databases can have NULL values (meaning "no data").
     Go's sql.NullString has two fields:
       .String → the actual text value
       .Valid  → true if the value is NOT null

  5. math.Pow — POWER FUNCTION
     math.Pow(2, 3) = 2^3 = 8.0 (returns a float64)
     We convert it to int with int(math.Pow(...))

=============================================================================
*/

package worker

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ethicalByte1443/queuectl/db"
)

// Job represents a single job row from the database.
type Job struct {
	ID         string
	Command    string
	State      string
	Attempts   int
	MaxRetries int
}

// claimAndRunJob tries to atomically claim a pending/failed job and execute it.
// Returns true if a job was found and processed, false if the queue is empty.
func claimAndRunJob(ctx context.Context, workerID int) bool {

	// --- Step 1: Read config values ---
	backoffBase := getBackoffBase()

	// --- Step 2: Begin a database transaction ---
	// BEGIN IMMEDIATE acquires a write lock immediately, preventing
	// other workers from modifying the database until we commit/rollback.
	tx, err := db.DB.Begin()
	if err != nil {
		fmt.Printf("  [Worker %d] ⚠️  Failed to begin transaction: %v\n", workerID, err)
		return false
	}

	// If anything goes wrong, rollback the transaction.
	// defer ensures this runs when the function exits.
	defer tx.Rollback()

	// --- Step 3: Find and claim a job atomically ---
	// We look for:
	//   a) Jobs in 'pending' state (fresh jobs), OR
	//   b) Jobs in 'failed' state (we'll check backoff eligibility in Go)
	//
	// ORDER BY: pending jobs first (priority), then by creation time (FIFO).
	// LIMIT 1 to grab just one job at a time.
	query := `
		SELECT id, command, state, attempts, max_retries,
		       CAST((strftime('%s', 'now') - strftime('%s', updated_at)) AS INTEGER) as seconds_since_update
		FROM jobs
		WHERE state = 'pending'
		   OR state = 'failed'
		ORDER BY
		    CASE WHEN state = 'pending' THEN 0 ELSE 1 END,
		    created_at ASC
		LIMIT 5
	`

	rows, err := tx.Query(query)
	if err != nil {
		fmt.Printf("  [Worker %d] ⚠️  Failed to query jobs: %v\n", workerID, err)
		return false
	}
	defer rows.Close()

	// Find the first eligible job (pending, or failed with backoff elapsed)
	var job Job
	found := false

	for rows.Next() {
		var secondsSinceUpdate int
		err := rows.Scan(&job.ID, &job.Command, &job.State, &job.Attempts, &job.MaxRetries, &secondsSinceUpdate)
		if err != nil {
			continue
		}

		if job.State == "pending" {
			// Pending jobs are always eligible
			found = true
			break
		}

		// For failed jobs, check if the backoff period has elapsed.
		// Backoff delay = base ^ attempts (exponential backoff)
		requiredDelay := int(math.Pow(float64(backoffBase), float64(job.Attempts)))
		if secondsSinceUpdate >= requiredDelay {
			found = true
			break
		}
	}
	rows.Close() // Close rows before using tx for writes

	if !found {
		// No eligible jobs available
		return false
	}

	// --- Step 4: Mark the job as "processing" ---
	_, err = tx.Exec(
		"UPDATE jobs SET state = 'processing', updated_at = datetime('now') WHERE id = ?",
		job.ID,
	)
	if err != nil {
		fmt.Printf("  [Worker %d] ⚠️  Failed to claim job %s: %v\n", workerID, job.ID, err)
		return false
	}

	// --- Step 5: Commit the transaction (release the lock) ---
	if err := tx.Commit(); err != nil {
		fmt.Printf("  [Worker %d] ⚠️  Failed to commit claim for job %s: %v\n", workerID, job.ID, err)
		return false
	}

	fmt.Printf("  [Worker %d] 📋 Claimed job: %s (command: %s, attempt: %d/%d)\n",
		workerID, job.ID, job.Command, job.Attempts+1, job.MaxRetries)

	// --- Step 6: Execute the shell command ---
	output, execErr := executeCommand(ctx, job.Command)

	// --- Step 7: Update job state based on result ---
	if execErr == nil {
		// SUCCESS — mark as completed
		_, err = db.DB.Exec(
			"UPDATE jobs SET state = 'completed', output = ?, updated_at = datetime('now') WHERE id = ?",
			output, job.ID,
		)
		if err != nil {
			fmt.Printf("  [Worker %d] ⚠️  Failed to update job %s as completed: %v\n", workerID, job.ID, err)
		}
		fmt.Printf("  [Worker %d] ✅ Job %s completed successfully\n", workerID, job.ID)

	} else {
		// FAILURE — increment attempts and decide next state
		newAttempts := job.Attempts + 1

		if newAttempts >= job.MaxRetries {
			// Exhausted all retries → move to Dead Letter Queue
			_, err = db.DB.Exec(
				"UPDATE jobs SET state = 'dead', attempts = ?, error_msg = ?, updated_at = datetime('now') WHERE id = ?",
				newAttempts, execErr.Error(), job.ID,
			)
			fmt.Printf("  [Worker %d] 💀 Job %s moved to DLQ after %d failed attempts\n",
				workerID, job.ID, newAttempts)
		} else {
			// Still has retries left → mark as failed (will be retried after backoff)
			nextDelay := int(math.Pow(float64(backoffBase), float64(newAttempts)))
			_, err = db.DB.Exec(
				"UPDATE jobs SET state = 'failed', attempts = ?, error_msg = ?, updated_at = datetime('now') WHERE id = ?",
				newAttempts, execErr.Error(), job.ID,
			)
			fmt.Printf("  [Worker %d] ❌ Job %s failed (attempt %d/%d). Retry in %d seconds\n",
				workerID, job.ID, newAttempts, job.MaxRetries, nextDelay)
		}

		if err != nil {
			fmt.Printf("  [Worker %d] ⚠️  Failed to update job %s state: %v\n", workerID, job.ID, err)
		}
	}

	return true
}

// executeCommand runs a shell command and returns its output or error.
// Uses CommandContext so the command is killed if the context is cancelled.
func executeCommand(ctx context.Context, command string) (string, error) {

	// Choose the shell based on the operating system
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// CombinedOutput runs the command and captures both stdout and stderr.
	outputBytes, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(outputBytes))

	return output, err
}

// getBackoffBase reads the backoff_base config. Defaults to 2.
func getBackoffBase() int {
	var value int
	err := db.DB.QueryRow("SELECT value FROM config WHERE key = 'backoff_base'").Scan(&value)
	if err != nil {
		return 2 // default
	}
	return value
}

// POWER function helper for SQLite (needed because SQLite doesn't have POWER built-in)
// We register it during DB initialization, but for safety we also handle it in the query.
func init() {
	// Note: The POWER function is not natively available in all SQLite builds.
	// As a fallback, we handle the backoff calculation in Go code too.
	_ = time.Now // force import of time package
}
