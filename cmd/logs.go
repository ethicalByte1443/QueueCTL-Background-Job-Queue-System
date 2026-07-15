package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [job_id]",
	Short: "View output logs of a job",
	Long:  `Show stdout/stderr output and any error messages recorded for a job.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobID := args[0]

		var command, state, output, errorMsg, createdAt, updatedAt string
		var attempts, maxRetries, durationMs int

		query := `SELECT command, state, output, error_msg, attempts, max_retries, duration_ms, created_at, updated_at
		          FROM jobs WHERE id = ?`

		err := db.DB.QueryRow(query, jobID).Scan(
			&command, &state, &output, &errorMsg, &attempts, &maxRetries, &durationMs, &createdAt, &updatedAt,
		)

		if err == sql.ErrNoRows {
			fmt.Fprintf(os.Stderr, "[ERROR] Job '%s' not found.\n", jobID)
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to query job logs: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n==============================================================")
		fmt.Printf("Job Details: %s\n", jobID)
		fmt.Println("==============================================================")
		fmt.Printf("  Command:    %s\n", command)
		fmt.Printf("  State:      %s\n", state)
		fmt.Printf("  Attempts:   %d/%d\n", attempts, maxRetries)
		fmt.Printf("  Duration:   %d ms\n", durationMs)
		fmt.Printf("  Created At: %s\n", createdAt)
		fmt.Printf("  Updated At: %s\n", updatedAt)
		fmt.Println("==============================================================")

		if errorMsg != "" {
			fmt.Println("Error Message:")
			fmt.Println(errorMsg)
			fmt.Println("==============================================================")
		}

		fmt.Println("Execution Output:")
		if output != "" {
			fmt.Println(output)
		} else {
			fmt.Println("(No output was captured)")
		}
		fmt.Println("==============================================================")
	},
}
