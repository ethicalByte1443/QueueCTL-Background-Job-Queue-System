/*
=============================================================================
🎓 LEARNING NOTE — cmd/status.go (Full Implementation)
=============================================================================

WHAT THIS FILE DOES:
  Queries the database and prints an aggregate summary:
    Pending:    5
    Processing: 2
    Completed:  10
    Failed:     1
    Dead (DLQ): 3
    ─────────────
    Total:      21

KEY GO CONCEPT — GROUP BY + COUNT:
  SQL's "GROUP BY state" groups all rows by their state column,
  then COUNT(*) counts how many rows are in each group.
  Result might look like:
    state     | count
    ----------+------
    pending   | 5
    completed | 10
    dead      | 3

KEY GO CONCEPT — rows.Next() LOOP:
  When a SQL query returns multiple rows, you iterate with:
    for rows.Next() {
        rows.Scan(&var1, &var2)  // extract columns into variables
    }
  Always call rows.Close() when done (or use defer).

=============================================================================
*/

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

		// Collect counts into a map: state → count
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

		// Print formatted output
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
