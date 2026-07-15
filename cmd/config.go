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
  queuectl config set max-retries 3     # Set max retries using positional command`,
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

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set configuration parameter",
	Long: `Set a specific configuration parameter for the queue system.
Available keys:
  max-retries (or max_retries)
  backoff-base (or backoff_base)

Example:
  queuectl config set max-retries 3
  queuectl config set backoff-base 2`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		valStr := args[1]

		var val int
		if _, err := fmt.Sscanf(valStr, "%d", &val); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Invalid value '%s': must be an integer\n", valStr)
			os.Exit(1)
		}

		dbKey := ""
		switch key {
		case "max-retries", "max_retries":
			dbKey = "max_retries"
		case "backoff-base", "backoff_base":
			dbKey = "backoff_base"
		default:
			fmt.Fprintf(os.Stderr, "[ERROR] Invalid config key '%s'. Supported keys: max-retries, backoff-base\n", key)
			os.Exit(1)
		}

		if err := setConfig(dbKey, val); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to save config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("[SUCCESS] Config parameter '%s' updated to %d\n", key, val)
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
	configCmd.AddCommand(configSetCmd)
	configCmd.Flags().IntVar(&maxRetries, "max-retries", 3,
		"Maximum number of retry attempts before moving a job to the DLQ")
	configCmd.Flags().IntVar(&backoffBase, "backoff-base", 2,
		"Base value (in seconds) for exponential backoff calculation")
}
