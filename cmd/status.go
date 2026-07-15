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
	Long: `Display a summary of all jobs grouped by their current state.

Example:
  qcli status`,
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

		fmt.Println("Queue Status")
		fmt.Println("  ──────────────────")
		fmt.Printf("  Pending:    %d\n", counts["pending"])
		fmt.Printf("  Processing: %d\n", counts["processing"])
		fmt.Printf("  Completed:  %d\n", counts["completed"])
		fmt.Printf("  Failed:     %d\n", counts["failed"])
		fmt.Printf("  Dead (DLQ): %d\n", counts["dead"])
		fmt.Println("  ──────────────────")
		fmt.Printf("  Total:      %d\n", total)
	},
}
