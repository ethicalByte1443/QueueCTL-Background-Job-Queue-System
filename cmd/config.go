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
  qcli config                       # Show current config
  qcli config --max-retries 5       # Set max retries to 5
  qcli config --backoff-base 3      # Set backoff base to 3 seconds`,
	Run: func(cmd *cobra.Command, args []string) {
		maxRetriesChanged := cmd.Flags().Changed("max-retries")
		backoffBaseChanged := cmd.Flags().Changed("backoff-base")

		if maxRetriesChanged {
			if err := setConfig("max_retries", maxRetries); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to save max-retries: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[SUCCESS] max-retries set to %d\n", maxRetries)
		}

		if backoffBaseChanged {
			if err := setConfig("backoff_base", backoffBase); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to save backoff-base: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[SUCCESS] backoff-base set to %d seconds\n", backoffBase)
		}

		fmt.Println("\nCurrent Configuration:")
		fmt.Printf("  Max Retries:  %d\n", getConfigInt("max_retries", 3))
		fmt.Printf("  Backoff Base: %d seconds\n", getConfigInt("backoff_base", 2))
	},
}

func setConfig(key string, value int) error {
	query := `
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`
	_, err := db.DB.Exec(query, key, value)
	return err
}

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
