/*
=============================================================================
🎓 LEARNING NOTE — cmd/config.go (Full Implementation)
=============================================================================

WHAT THIS FILE DOES:
  Implements the config command that stores/retrieves settings in SQLite.
  Settings are stored in a simple key-value "config" table.

KEY GO CONCEPT — UPSERT:
  "UPSERT" = UPDATE + INSERT. It means:
    "INSERT this row, but if a row with the same key already exists,
     UPDATE it instead of throwing an error."

  In SQLite, this is done with:
    INSERT ... ON CONFLICT(key) DO UPDATE SET value = excluded.value

  "excluded.value" refers to the value we tried to insert but conflicted.

=============================================================================
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
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
		// Check if any flags were explicitly set by the user.
		// cmd.Flags().Changed() returns true only if the user typed the flag.
		maxRetriesChanged := cmd.Flags().Changed("max-retries")
		backoffBaseChanged := cmd.Flags().Changed("backoff-base")

		// If user provided flags, save them
		if maxRetriesChanged {
			if err := setConfig("max_retries", maxRetries); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to save max-retries: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ max-retries set to %d\n", maxRetries)
		}

		if backoffBaseChanged {
			if err := setConfig("backoff_base", backoffBase); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to save backoff-base: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ backoff-base set to %d seconds\n", backoffBase)
		}

		// Always show current config
		fmt.Println("\n⚙️  Current Configuration:")
		fmt.Printf("  Max Retries:  %d\n", getConfigInt("max_retries", 3))
		fmt.Printf("  Backoff Base: %d seconds\n", getConfigInt("backoff_base", 2))
	},
}

// setConfig saves a key-value pair in the config table.
// Uses UPSERT to insert or update if the key already exists.
func setConfig(key string, value int) error {
	query := `
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`
	_, err := db.DB.Exec(query, key, value)
	return err
}

// getConfigInt reads an integer config value from the database.
// Returns the defaultVal if the key is not found.
func getConfigInt(key string, defaultVal int) int {
	var value int
	err := db.DB.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err != nil {
		return defaultVal
	}
	return value
}

func init() {
	configCmd.Flags().IntVar(&maxRetries, "max-retries", 3,
		"Maximum number of retry attempts before moving a job to the DLQ")
	configCmd.Flags().IntVar(&backoffBase, "backoff-base", 2,
		"Base value (in seconds) for exponential backoff calculation")
}
