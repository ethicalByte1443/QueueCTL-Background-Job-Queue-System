/*
=============================================================================
🎓 LEARNING NOTE — cmd/config.go (Config Subcommand)
=============================================================================

WHAT THIS FILE DOES:
  Defines the "config" subcommand for setting runtime parameters.
  Usage:
    queuectl config --max-retries 5
    queuectl config --backoff-base 3

  This lets users customize how the queue behaves without editing code.

KEY CONCEPT — WHY RUNTIME CONFIG?
  Different jobs may need different retry strategies. A quick API call
  might need 5 retries with a 2-second base, while a heavy computation
  might need 2 retries with a 10-second base. The config command lets
  you tune these without recompiling the application.

=============================================================================
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	maxRetries  int
	backoffBase int
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or update queue configuration",
	Long: `View or set configuration parameters for the job queue system.

Examples:
  queuectl config                       # Show current config
  queuectl config --max-retries 5       # Set max retries to 5
  queuectl config --backoff-base 3      # Set backoff base to 3 seconds`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Task 4 will implement config storage/retrieval
		fmt.Println("⚙️  Queue Configuration")
		fmt.Printf("  Max Retries:  %d\n", maxRetries)
		fmt.Printf("  Backoff Base: %d seconds\n", backoffBase)
	},
}

func init() {
	configCmd.Flags().IntVar(&maxRetries, "max-retries", 3,
		"Maximum number of retry attempts before moving a job to the DLQ")
	configCmd.Flags().IntVar(&backoffBase, "backoff-base", 2,
		"Base value (in seconds) for exponential backoff calculation")
}
