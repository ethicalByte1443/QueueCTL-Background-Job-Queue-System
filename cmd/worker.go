/*
=============================================================================
🎓 LEARNING NOTE — cmd/worker.go (Worker Subcommand)
=============================================================================

WHAT THIS FILE DOES:
  Defines the "worker" command group with two sub-subcommands:
    queuectl worker start --count 3
    queuectl worker stop

KEY GO CONCEPT — NESTED COMMANDS (Parent → Child):

  "worker" is a PARENT command that doesn't do anything by itself.
  "start" and "stop" are CHILDREN of "worker".

  This creates a natural command hierarchy:
    queuectl worker start --count 3
    queuectl worker stop

KEY GO CONCEPT — FLAGS:

  Flags are the --options in CLI commands (like --count 3).
  We use cmd.Flags().IntVarP() to define a flag:
    - &workerCount  → pointer to the variable that stores the value
    - "count"       → the long flag name (--count)
    - "c"           → the short flag name (-c)
    - 1             → the default value
    - "description" → help text

=============================================================================
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// workerCount stores the value from --count flag. Default is 1 worker.
var workerCount int

// workerCmd is the PARENT command: "queuectl worker"
// It doesn't do anything by itself — it just groups "start" and "stop".
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage background workers",
	Long:  `Start or stop background worker processes that pick up and execute queued jobs.`,
}

// workerStartCmd is the "queuectl worker start" command.
var workerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start background workers",
	Long: `Start one or more background worker processes.

Workers continuously poll the queue for pending jobs, execute them,
and handle retries with exponential backoff.

Example:
  queuectl worker start           # Start 1 worker (default)
  queuectl worker start --count 3 # Start 3 workers in parallel`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Task 5 will implement the actual worker pool here
		fmt.Printf("🚀 Starting %d worker(s)...\n", workerCount)
	},
}

// workerStopCmd is the "queuectl worker stop" command.
var workerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Gracefully stop all running workers",
	Long: `Send a stop signal to all running workers. Workers will finish
their current job before shutting down (graceful shutdown).`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Task 5 will implement graceful shutdown here
		fmt.Println("🛑 Stopping workers...")
	},
}

func init() {
	// Register "start" and "stop" as children of "worker"
	workerCmd.AddCommand(workerStartCmd)
	workerCmd.AddCommand(workerStopCmd)

	// Add the --count / -c flag to the "start" subcommand only
	workerStartCmd.Flags().IntVarP(&workerCount, "count", "c", 1,
		"Number of worker goroutines to start")
}
