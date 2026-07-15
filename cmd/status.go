/*
=============================================================================
🎓 LEARNING NOTE — cmd/status.go (Status Subcommand)
=============================================================================

WHAT THIS FILE DOES:
  Defines the "status" subcommand. When the user types:
    queuectl status
  It will show a summary like:
    Pending:    5
    Processing: 2
    Completed:  10
    Dead (DLQ): 1

=============================================================================
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show queue statistics",
	Long: `Display a summary of all jobs grouped by their current state.

Example:
  queuectl status`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Task 6 will query the DB and print real stats
		fmt.Println("📊 Queue Status")
		fmt.Println("  Pending:    -")
		fmt.Println("  Processing: -")
		fmt.Println("  Completed:  -")
		fmt.Println("  Failed:     -")
		fmt.Println("  Dead (DLQ): -")
	},
}
