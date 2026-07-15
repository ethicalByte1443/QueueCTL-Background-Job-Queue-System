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
