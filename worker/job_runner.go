package worker

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ethicalByte1443/queuectl/db"
)

type Job struct {
	ID         string
	Command    string
	State      string
	Attempts   int
	MaxRetries int
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
		fmt.Printf("  [Worker %d] [WARNING] Failed to query jobs: %v\n", workerID, err)
		return false
	}
	defer rows.Close()

	var job Job
	found := false

	for rows.Next() {
		var secondsSinceUpdate int
		err := rows.Scan(&job.ID, &job.Command, &job.State, &job.Attempts, &job.MaxRetries, &secondsSinceUpdate)
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

	_, err = tx.Exec(
		"UPDATE jobs SET state = 'processing', updated_at = datetime('now') WHERE id = ?",
		job.ID,
	)
	if err != nil {
		fmt.Printf("  [Worker %d] [WARNING] Failed to claim job %s: %v\n", workerID, job.ID, err)
		return false
	}

	if err := tx.Commit(); err != nil {
		fmt.Printf("  [Worker %d] [WARNING] Failed to commit claim for job %s: %v\n", workerID, job.ID, err)
		return false
	}

	fmt.Printf("  [Worker %d] [INFO] Claimed job: %s (command: %s, attempt: %d/%d)\n",
		workerID, job.ID, job.Command, job.Attempts+1, job.MaxRetries)

	output, execErr := executeCommand(ctx, job.Command)

	if execErr == nil {
		_, err = db.DB.Exec(
			"UPDATE jobs SET state = 'completed', output = ?, updated_at = datetime('now') WHERE id = ?",
			output, job.ID,
		)
		if err != nil {
			fmt.Printf("  [Worker %d] [WARNING] Failed to update job %s as completed: %v\n", workerID, job.ID, err)
		}
		fmt.Printf("  [Worker %d] [SUCCESS] Job %s completed successfully\n", workerID, job.ID)

	} else {
		newAttempts := job.Attempts + 1

		if newAttempts >= job.MaxRetries {
			_, err = db.DB.Exec(
				"UPDATE jobs SET state = 'dead', attempts = ?, error_msg = ?, updated_at = datetime('now') WHERE id = ?",
				newAttempts, execErr.Error(), job.ID,
			)
			fmt.Printf("  [Worker %d] [DLQ] Job %s moved to DLQ after %d failed attempts\n",
				workerID, job.ID, newAttempts)
		} else {
			nextDelay := int(math.Pow(float64(backoffBase), float64(newAttempts)))
			_, err = db.DB.Exec(
				"UPDATE jobs SET state = 'failed', attempts = ?, error_msg = ?, updated_at = datetime('now') WHERE id = ?",
				newAttempts, execErr.Error(), job.ID,
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
