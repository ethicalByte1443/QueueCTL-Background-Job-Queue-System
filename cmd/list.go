/*
=============================================================================
🎓 LEARNING NOTE — cmd/list.go (List Subcommand)
=============================================================================

WHAT THIS FILE DOES:
  Defines the "list" subcommand. When the user types:
    queuectl list                  # list all jobs
    queuectl list --state pending  # list only pending jobs
  It will print individual job details.

KEY GO CONCEPT — STRING FLAGS:
  Similar to --count (integer flag), here we use StringVarP for --state,
  which accepts text values like "pending", "completed", "dead".

=============================================================================
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listState stores the --state flag value. Empty string means "show all".
var listState string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs in the queue",
	Long: `List all jobs or filter by state.

Examples:
  queuectl list                   # Show all jobs
  queuectl list --state pending   # Show only pending jobs
  queuectl list --state completed # Show only completed jobs`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Task 6 will query the DB and print real job details
		if listState != "" {
			fmt.Printf("📋 Listing jobs with state: %s\n", listState)
		} else {
			fmt.Println("📋 Listing all jobs...")
		}
	},
}

func init() {
	listCmd.Flags().StringVarP(&listState, "state", "s", "",
		"Filter jobs by state (pending, processing, completed, failed, dead)")
}
