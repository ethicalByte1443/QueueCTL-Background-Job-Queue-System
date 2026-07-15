/*
=============================================================================
🎓 LEARNING NOTE — cmd/worker.go (Worker Subcommand - Connected)
=============================================================================

WHAT THIS FILE DOES:
  Now connects the CLI commands to the actual worker package:
    queuectl worker start --count 3  → calls worker.StartWorkers(3)
    queuectl worker stop             → placeholder (workers stop via Ctrl+C)

  In this design, workers run IN THE FOREGROUND of the current terminal.
  You stop them by pressing Ctrl+C. The "stop" command is a placeholder
  for a future feature where workers could run as background daemons.

=============================================================================
*/

package cmd

import (
	"fmt"

	"github.com/ethicalByte1443/queuectl/worker"
	"github.com/spf13/cobra"
)

var workerCount int

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage background workers",
	Long:  `Start or stop background worker processes that pick up and execute queued jobs.`,
}

var workerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start background workers",
	Long: `Start one or more background worker processes.

Workers continuously poll the queue for pending jobs, execute them,
and handle retries with exponential backoff.

Example:
  qcli worker start           # Start 1 worker (default)
  qcli worker start --count 3 # Start 3 workers in parallel`,
	Run: func(cmd *cobra.Command, args []string) {
		// This calls the real worker pool implementation.
		// It blocks until Ctrl+C is pressed.
		worker.StartWorkers(workerCount)
	},
}

var workerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Gracefully stop all running workers",
	Long: `Workers are stopped by pressing Ctrl+C in the terminal where
they are running. This sends a graceful shutdown signal.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[INFO] Workers run in the foreground and can be stopped with Ctrl+C.")
		fmt.Println("       Start workers with: qcli worker start --count 3")
	},
}

func init() {
	workerCmd.AddCommand(workerStartCmd)
	workerCmd.AddCommand(workerStopCmd)

	workerStartCmd.Flags().IntVarP(&workerCount, "count", "c", 1,
		"Number of worker goroutines to start")
}
