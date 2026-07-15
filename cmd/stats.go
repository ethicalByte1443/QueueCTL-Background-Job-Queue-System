package cmd

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show performance and execution metrics",
	Long:  `Display queue throughput, execution times, success rates, and active worker stats.`,
	Run: func(cmd *cobra.Command, args []string) {
		var totalJobs, completedJobs, failedJobs, deadJobs, processingJobs, pendingJobs int

		// Count jobs per state
		rows, err := db.DB.Query("SELECT state, COUNT(*) FROM jobs GROUP BY state")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to query job statistics: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		for rows.Next() {
			var state string
			var count int
			if err := rows.Scan(&state, &count); err == nil {
				totalJobs += count
				switch state {
				case "completed":
					completedJobs = count
				case "failed":
					failedJobs = count
				case "dead":
					deadJobs = count
				case "processing":
					processingJobs = count
				case "pending":
					pendingJobs = count
				}
			}
		}

		// Query execution time statistics for completed jobs
		var avgDuration, minDuration, maxDuration float64
		err = db.DB.QueryRow(`
			SELECT COALESCE(AVG(duration_ms), 0), COALESCE(MIN(duration_ms), 0), COALESCE(MAX(duration_ms), 0)
			FROM jobs
			WHERE state = 'completed'
		`).Scan(&avgDuration, &minDuration, &maxDuration)
		if err != nil {
			// Ignore if failed or no completed jobs
		}

		// Query active worker processes count
		var activeWorkersCount, activeGoroutinesCount int
		err = db.DB.QueryRow(`
			SELECT COUNT(*), COALESCE(SUM(worker_count), 0)
			FROM worker_processes
			WHERE status = 'running'
			  AND updated_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-10 seconds')
		`).Scan(&activeWorkersCount, &activeGoroutinesCount)
		if err != nil {
			// Ignore
		}

		// Calculate rates
		successRate := 0.0
		if totalJobs > 0 {
			successRate = (float64(completedJobs) / float64(totalJobs)) * 100.0
		}

		fmt.Println("\nQueue CTL - System Performance Metrics")
		fmt.Println("==============================================================")
		fmt.Printf("Job Counts Summary:\n")
		fmt.Printf("  Total Enqueued Jobs:   %d\n", totalJobs)
		fmt.Printf("  Pending Jobs:          %d\n", pendingJobs)
		fmt.Printf("  Processing Jobs:       %d\n", processingJobs)
		fmt.Printf("  Completed Jobs:        %d\n", completedJobs)
		fmt.Printf("  Failed (Retryable):    %d\n", failedJobs)
		fmt.Printf("  Dead Letter (DLQ):     %d\n", deadJobs)
		fmt.Println("--------------------------------------------------------------")
		fmt.Printf("Efficiency & Reliability:\n")
		fmt.Printf("  Success Rate:          %.2f%%\n", successRate)
		if completedJobs > 0 {
			fmt.Printf("  Avg Execution Time:    %.2f ms\n", avgDuration)
			fmt.Printf("  Min Execution Time:    %.2f ms\n", minDuration)
			fmt.Printf("  Max Execution Time:    %.2f ms\n", maxDuration)
		} else {
			fmt.Printf("  Avg Execution Time:    N/A (No completed jobs)\n")
		}
		fmt.Println("--------------------------------------------------------------")
		fmt.Printf("Active Infrastructure:\n")
		fmt.Printf("  Worker Processes:      %d\n", activeWorkersCount)
		fmt.Printf("  Worker Goroutines:     %d\n", activeGoroutinesCount)
		fmt.Println("==============================================================")
	},
}
