package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "qcli",
	Short: "A CLI-based background job queue system",
	Long: `QCli is a background job queue system that manages tasks with
worker processes, automatic retries using exponential backoff,
and a Dead Letter Queue (DLQ) for permanently failed jobs.

Usage examples:
  qcli enqueue --id job1 --command "echo hello"
  qcli worker start --count 3
  qcli status
  qcli dlq list`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(enqueueCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(dlqCmd)
	rootCmd.AddCommand(configCmd)
}
