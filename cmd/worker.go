package cmd

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
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
  queuectl worker start           # Start 1 worker (default)
  queuectl worker start --count 3 # Start 3 workers in parallel`,
	Run: func(cmd *cobra.Command, args []string) {
		worker.StartWorkers(workerCount)
	},
}

var workerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Gracefully stop all running workers",
	Long: `Sends a graceful shutdown signal to all active worker processes in the database.
Each worker process will finish its currently executing jobs before shutting down.`,
	Run: func(cmd *cobra.Command, args []string) {
		result, err := db.DB.Exec("UPDATE worker_processes SET status = 'stop_requested' WHERE status = 'running'")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to send stop signal: %v\n", err)
			os.Exit(1)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			fmt.Println("[INFO] No running worker processes found.")
			return
		}

		fmt.Printf("[INFO] Sent graceful stop signal to %d running worker process(es).\n", rowsAffected)
		fmt.Println("       Workers will finish their current jobs and stop gracefully.")
	},
}

func init() {
	workerCmd.AddCommand(workerStartCmd)
	workerCmd.AddCommand(workerStopCmd)

	workerStartCmd.Flags().IntVarP(&workerCount, "count", "c", 1,
		"Number of worker goroutines to start")
}
