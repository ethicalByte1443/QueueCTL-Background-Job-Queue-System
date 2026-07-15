/*
=============================================================================
🎓 LEARNING NOTE — cmd/list.go (Full Implementation)
=============================================================================

WHAT THIS FILE DOES:
  Lists individual jobs with their details. Can filter by state.
    queuectl list                   → show all jobs
    queuectl list --state pending   → show only pending jobs
    queuectl list --state dead      → show only dead (DLQ) jobs

KEY GO CONCEPT — CONDITIONAL SQL BUILDING:
  We dynamically build the SQL query based on whether --state was provided.
  If the user passes --state, we add a WHERE clause. If not, we select all.

KEY GO CONCEPT — fmt.Sprintf for FORMATTING:
  %-8s  → left-aligned string, padded to 8 characters
  %-12s → left-aligned string, padded to 12 characters
  This creates a nicely aligned table in the terminal output.

=============================================================================
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

var listState string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs in the queue",
	Long: `List all jobs or filter by state.

Examples:
  qcli list                   # Show all jobs
  qcli list --state pending   # Show only pending jobs
  qcli list --state completed # Show only completed jobs`,
	Run: func(cmd *cobra.Command, args []string) {
		var query string
		var queryArgs []interface{}

		if listState != "" {
			query = `SELECT id, command, state, attempts, max_retries, error_msg, created_at, updated_at
			         FROM jobs WHERE state = ? ORDER BY created_at DESC`
			queryArgs = append(queryArgs, listState)
		} else {
			query = `SELECT id, command, state, attempts, max_retries, error_msg, created_at, updated_at
			         FROM jobs ORDER BY created_at DESC`
		}

		rows, err := db.DB.Query(query, queryArgs...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to query jobs: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		// Print header
		fmt.Printf("\n%-14s %-25s %-12s %-10s %-30s\n",
			"ID", "COMMAND", "STATE", "ATTEMPTS", "ERROR")
		fmt.Println("──────────────────────────────────────────────────────────────────────────────────────────")

		count := 0
		for rows.Next() {
			var id, command, state, errorMsg, createdAt, updatedAt string
			var attempts, maxRetries int

			if err := rows.Scan(&id, &command, &state, &attempts, &maxRetries, &errorMsg, &createdAt, &updatedAt); err != nil {
				continue
			}

			// Truncate long commands/errors for display
			if len(command) > 23 {
				command = command[:20] + "..."
			}
			if len(errorMsg) > 28 {
				errorMsg = errorMsg[:25] + "..."
			}
			if errorMsg == "" {
				errorMsg = "-"
			}

			attemptsStr := fmt.Sprintf("%d/%d", attempts, maxRetries)

			fmt.Printf("%-14s %-25s %-12s %-10s %-30s\n",
				id, command, state, attemptsStr, errorMsg)
			count++
		}

		if count == 0 {
			if listState != "" {
				fmt.Printf("\n  No jobs with state '%s'\n", listState)
			} else {
				fmt.Println("\n  No jobs in the queue")
			}
		}
		fmt.Printf("\nTotal: %d job(s)\n", count)
	},
}

func init() {
	listCmd.Flags().StringVarP(&listState, "state", "s", "",
		"Filter jobs by state (pending, processing, completed, failed, dead)")
}
