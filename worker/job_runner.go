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

type Job struct {
	ID         string
	Command    string
	State      string
	Attempts   int
	MaxRetries int
	Timeout    int
}

func claimAndRunJob(ctx context.Context, workerID int) bool {
	backoffBase := getBackoffBase()

	tx, err := db.DB.Begin()
	if err != nil {
		fmt.Printf("  [Worker %d] [WARNING] Failed to begin transaction: %v\n", workerID, err)
		return false
	}
	defer tx.Rollback()

	query := `
		SELECT id, command, state, attempts, max_retries, timeout,
		       CAST((strftime('%s', 'now') - strftime('%s', updated_at)) AS INTEGER) as seconds_since_update
		FROM jobs
		WHERE (state = 'pending' OR state = 'failed')
		  AND (run_at IS NULL OR run_at <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ORDER BY
		    CASE WHEN state = 'pending' THEN 0 ELSE 1 END,
		    priority DESC,
		    created_at ASC
		LIMIT 5
	`

	rows, err := tx.Query(query)
	if err != nil {
		fmt.Printf("  [Worker %d] [WARNING] Failed to query jobs: %v\n", workerID, err)
		return false
	}
	defer rows.Close()

	var job Job
	found := false

	for rows.Next() {
		var secondsSinceUpdate int
		err := rows.Scan(&job.ID, &job.Command, &job.State, &job.Attempts, &job.MaxRetries, &job.Timeout, &secondsSinceUpdate)
		if err != nil {
			continue
		}

		if job.State == "pending" {
			found = true
			break
		}

		requiredDelay := int(math.Pow(float64(backoffBase), float64(job.Attempts)))
		if secondsSinceUpdate >= requiredDelay {
			found = true
			break
		}
	}
	rows.Close()

	if !found {
		return false
	}

	// Lock the job to prevent duplicate execution by checking the state in the WHERE clause
	result, err := tx.Exec(
		"UPDATE jobs SET state = 'processing', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ? AND state = ?",
		job.ID, job.State,
	)
	if err != nil {
		fmt.Printf("  [Worker %d] [WARNING] Failed to claim job %s: %v\n", workerID, job.ID, err)
		return false
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		// Job already claimed by another worker process/thread
		return false
	}

	if err := tx.Commit(); err != nil {
		fmt.Printf("  [Worker %d] [WARNING] Failed to commit claim for job %s: %v\n", workerID, job.ID, err)
		return false
	}

	fmt.Printf("  [Worker %d] [INFO] Claimed job: %s (command: %s, attempt: %d/%d)\n",
		workerID, job.ID, job.Command, job.Attempts+1, job.MaxRetries)

	// Determine job execution timeout
	execTimeout := time.Duration(job.Timeout) * time.Second
	if execTimeout <= 0 {
		execTimeout = 600 * time.Second // 10 minutes default
	}

	// Use background context for job execution so worker graceful stop signaling does not abort running jobs
	cmdCtx, cmdCancel := context.WithTimeout(context.Background(), execTimeout)
	defer cmdCancel()

	startTime := time.Now()
	output, execErr := executeCommand(cmdCtx, job.Command)
	durationMs := time.Since(startTime).Milliseconds()

	if execErr == nil {
		_, err = db.DB.Exec(
			"UPDATE jobs SET state = 'completed', output = ?, duration_ms = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?",
			output, durationMs, job.ID,
		)
		if err != nil {
			fmt.Printf("  [Worker %d] [WARNING] Failed to update job %s as completed: %v\n", workerID, job.ID, err)
		}
		fmt.Printf("  [Worker %d] [SUCCESS] Job %s completed successfully\n", workerID, job.ID)

	} else {
		newAttempts := job.Attempts + 1

		if newAttempts >= job.MaxRetries {
			_, err = db.DB.Exec(
				"UPDATE jobs SET state = 'dead', attempts = ?, error_msg = ?, duration_ms = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?",
				newAttempts, execErr.Error(), durationMs, job.ID,
			)
			fmt.Printf("  [Worker %d] [DLQ] Job %s moved to DLQ after %d failed attempts\n",
				workerID, job.ID, newAttempts)
		} else {
			nextDelay := int(math.Pow(float64(backoffBase), float64(newAttempts)))
			_, err = db.DB.Exec(
				"UPDATE jobs SET state = 'failed', attempts = ?, error_msg = ?, duration_ms = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?",
				newAttempts, execErr.Error(), durationMs, job.ID,
			)
			fmt.Printf("  [Worker %d] [FAILED] Job %s failed (attempt %d/%d). Retry in %d seconds\n",
				workerID, job.ID, newAttempts, job.MaxRetries, nextDelay)
		}

		if err != nil {
			fmt.Printf("  [Worker %d] [WARNING] Failed to update job %s state: %v\n", workerID, job.ID, err)
		}
	}

	return true
}

func executeCommand(ctx context.Context, command string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	outputBytes, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(outputBytes))

	return output, err
}

func getBackoffBase() int {
	var value int
	err := db.DB.QueryRow("SELECT value FROM config WHERE key = 'backoff_base'").Scan(&value)
	if err != nil {
		return 2
	}
	return value
}
