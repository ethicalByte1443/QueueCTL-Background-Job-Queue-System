/*
=============================================================================
🎓 LEARNING NOTE — cmd/dlq.go (Dead Letter Queue Subcommand)
=============================================================================

WHAT THIS FILE DOES:
  Defines the "dlq" command group with two sub-subcommands:
    queuectl dlq list         # Show all dead/failed jobs
    queuectl dlq retry <id>   # Move a dead job back to pending for retry

  This is the same NESTED COMMAND pattern we used for "worker".

KEY CONCEPT — Dead Letter Queue:
  When a job fails more times than max_retries, it's moved to the "dead"
  state. The DLQ lets you inspect these failures and optionally retry them.
  "dlq retry" resets the job's state back to "pending" and its attempts to 0.

=============================================================================
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// dlqCmd is the PARENT command: "queuectl dlq"
var dlqCmd = &cobra.Command{
	Use:   "dlq",
	Short: "Manage the Dead Letter Queue",
	Long:  `View and retry jobs that have permanently failed after exhausting all retries.`,
}

// dlqListCmd is "queuectl dlq list"
var dlqListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all dead jobs",
	Long: `Display all jobs that have been moved to the Dead Letter Queue.

Example:
  queuectl dlq list`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Task 6 will query the DB for dead jobs
		fmt.Println("💀 Dead Letter Queue:")
		fmt.Println("  (No dead jobs yet)")
	},
}

// dlqRetryCmd is "queuectl dlq retry <job_id>"
var dlqRetryCmd = &cobra.Command{
	Use:   "retry [job_id]",
	Short: "Retry a dead job",
	Long: `Move a job from the Dead Letter Queue back to pending state.
The job's attempts counter will be reset to 0.

Example:
  queuectl dlq retry job1`,
	Args: cobra.ExactArgs(1), // Require exactly 1 argument (the job ID)
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Task 6 will implement retry logic
		fmt.Printf("🔄 Retrying dead job: %s\n", args[0])
	},
}

func init() {
	dlqCmd.AddCommand(dlqListCmd)
	dlqCmd.AddCommand(dlqRetryCmd)
}
