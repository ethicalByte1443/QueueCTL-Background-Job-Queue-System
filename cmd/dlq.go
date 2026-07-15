/*
=============================================================================
🎓 LEARNING NOTE — cmd/dlq.go (Full Implementation)
=============================================================================

WHAT THIS FILE DOES:
  Manages the Dead Letter Queue (DLQ):
    queuectl dlq list        → Show all dead jobs with error details
    queuectl dlq retry job1  → Move a dead job back to 'pending' for retry

KEY GO CONCEPT — STATE MACHINE TRANSITIONS:
  The DLQ retry command performs a state transition:
    dead → pending
  It also resets the attempts counter to 0, giving the job a fresh start.
  This is useful when you've fixed the underlying issue (e.g., installed
  a missing program) and want to re-process the failed job.

KEY GO CONCEPT — RowsAffected():
  After an UPDATE query, result.RowsAffected() tells you HOW MANY rows
  were actually changed. If it returns 0, it means the WHERE clause
  didn't match anything (e.g., the job ID doesn't exist or isn't dead).

=============================================================================
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

var dlqCmd = &cobra.Command{
	Use:   "dlq",
	Short: "Manage the Dead Letter Queue",
	Long:  `View and retry jobs that have permanently failed after exhausting all retries.`,
}

var dlqListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all dead jobs",
	Long: `Display all jobs in the Dead Letter Queue with their error details.

Example:
  qcli dlq list`,
	Run: func(cmd *cobra.Command, args []string) {
		query := `SELECT id, command, attempts, max_retries, error_msg, created_at, updated_at
		          FROM jobs WHERE state = 'dead' ORDER BY updated_at DESC`

		rows, err := db.DB.Query(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to query DLQ: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		fmt.Println("\nDead Letter Queue")
		fmt.Printf("%-14s %-25s %-10s %-40s\n",
			"ID", "COMMAND", "ATTEMPTS", "ERROR")
		fmt.Println("──────────────────────────────────────────────────────────────────────────────────────────")

		count := 0
		for rows.Next() {
			var id, command, errorMsg, createdAt, updatedAt string
			var attempts, maxRetries int

			if err := rows.Scan(&id, &command, &attempts, &maxRetries, &errorMsg, &createdAt, &updatedAt); err != nil {
				continue
			}

			if len(command) > 23 {
				command = command[:20] + "..."
			}
			if len(errorMsg) > 38 {
				errorMsg = errorMsg[:35] + "..."
			}

			attemptsStr := fmt.Sprintf("%d/%d", attempts, maxRetries)
			fmt.Printf("%-14s %-25s %-10s %-40s\n",
				id, command, attemptsStr, errorMsg)
			count++
		}

		if count == 0 {
			fmt.Println("\n  No dead jobs found in the DLQ.")
		}
		fmt.Printf("\nTotal: %d dead job(s)\n", count)
	},
}

var dlqRetryCmd = &cobra.Command{
	Use:   "retry [job_id]",
	Short: "Retry a dead job",
	Long: `Move a job from the Dead Letter Queue back to pending state.
The job's attempts counter will be reset to 0.

Example:
  qcli dlq retry job1`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobID := args[0]

		// Reset the job: state → pending, attempts → 0, clear error
		query := `UPDATE jobs
		          SET state = 'pending', attempts = 0, error_msg = '', updated_at = datetime('now')
		          WHERE id = ? AND state = 'dead'`

		result, err := db.DB.Exec(query, jobID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to retry job: %v\n", err)
			os.Exit(1)
		}

		// Check if the job was actually found and updated
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			fmt.Fprintf(os.Stderr, "[ERROR] Job '%s' not found in the DLQ. It might not exist or isn't in 'dead' state.\n", jobID)
			os.Exit(1)
		}

		fmt.Printf("[INFO] Job '%s' moved back to pending. It will be picked up by workers.\n", jobID)
	},
}

func init() {
	dlqCmd.AddCommand(dlqListCmd)
	dlqCmd.AddCommand(dlqRetryCmd)
}
