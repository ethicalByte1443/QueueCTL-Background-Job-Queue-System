package cmd

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show queue statistics",
	Long: `Display a summary of all jobs grouped by their current state and active workers info.

Example:
  queuectl status`,
	Run: func(cmd *cobra.Command, args []string) {
		query := `SELECT state, COUNT(*) as count FROM jobs GROUP BY state`

		rows, err := db.DB.Query(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to query status: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		counts := map[string]int{
			"pending":    0,
			"processing": 0,
			"completed":  0,
			"failed":     0,
			"dead":       0,
		}

		total := 0
		for rows.Next() {
			var state string
			var count int
			if err := rows.Scan(&state, &count); err != nil {
				continue
			}
			counts[state] = count
			total += count
		}

		// Query active worker processes
		workersQuery := `
			SELECT pid, worker_count, started_at
			FROM worker_processes
			WHERE status = 'running'
			  AND updated_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-10 seconds')
		`
		workerRows, err := db.DB.Query(workersQuery)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to query active workers: %v\n", err)
			os.Exit(1)
		}
		defer workerRows.Close()

		type WorkerProc struct {
			PID         int
			WorkerCount int
			StartedAt   string
		}

		var activeWorkers []WorkerProc
		totalWorkers := 0
		for workerRows.Next() {
			var w WorkerProc
			if err := workerRows.Scan(&w.PID, &w.WorkerCount, &w.StartedAt); err == nil {
				activeWorkers = append(activeWorkers, w)
				totalWorkers += w.WorkerCount
			}
		}

		fmt.Println("\nQueue Status")
		fmt.Println("  ────────────────────────────────────────")
		fmt.Printf("  Pending:       %d\n", counts["pending"])
		fmt.Printf("  Processing:    %d\n", counts["processing"])
		fmt.Printf("  Completed:     %d\n", counts["completed"])
		fmt.Printf("  Failed:        %d\n", counts["failed"])
		fmt.Printf("  Dead (DLQ):    %d\n", counts["dead"])
		fmt.Println("  ────────────────────────────────────────")
		fmt.Printf("  Total Jobs:    %d\n", total)

		fmt.Println("\nActive Workers")
		fmt.Println("  ────────────────────────────────────────")
		fmt.Printf("  Worker Processes:  %d\n", len(activeWorkers))
		fmt.Printf("  Total Goroutines:  %d\n", totalWorkers)

		if len(activeWorkers) > 0 {
			fmt.Println("\n  Running Processes:")
			fmt.Printf("    %-8s %-12s %-25s\n", "PID", "GOROUTINES", "STARTED AT")
			fmt.Println("    ────────────────────────────────────────────")
			for _, w := range activeWorkers {
				fmt.Printf("    %-8d %-12d %-25s\n", w.PID, w.WorkerCount, w.StartedAt)
			}
		}
		fmt.Println("")
	},
}
