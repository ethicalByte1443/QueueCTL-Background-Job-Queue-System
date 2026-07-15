package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

type JobPayload struct {
	Id       string `json:"id"`
	Command  string `json:"command"`
	Priority int    `json:"priority"`
	RunAt    string `json:"run_at"`
	Timeout  int    `json:"timeout"`
}

var (
	flagJobID    string
	flagCommand  string
	flagPriority int
	flagRunAt    string
	flagTimeout  int
)

var enqueueCmd = &cobra.Command{
	Use:   "enqueue [JSON payload]",
	Short: "Add a new job to the queue",
	Long: `Enqueue adds a new job to the processing queue.

You can either pass a valid JSON string with fields (id, command, priority, run_at, timeout),
or use the --id and --command flags directly.

Examples:
  queuectl enqueue --id job1 --command "echo hello world" --priority 5 --timeout 30
  queuectl enqueue '{"id":"job1", "command":"echo hello world", "priority": 10, "run_at": "2026-07-16T15:00:00Z"}'`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var payload JobPayload

		if flagJobID != "" || flagCommand != "" {
			if flagJobID == "" || flagCommand == "" {
				fmt.Fprintln(os.Stderr, "[ERROR] Both --id and --command flags must be provided if using flags.")
				os.Exit(1)
			}
			payload.Id = flagJobID
			payload.Command = flagCommand
			payload.Priority = flagPriority
			payload.RunAt = flagRunAt
			payload.Timeout = flagTimeout
		} else {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "[ERROR] Must provide either a JSON payload argument or the --id and --command flags.")
				_ = cmd.Help()
				os.Exit(1)
			}

			jsonInput := args[0]
			if err := json.Unmarshal([]byte(jsonInput), &payload); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Invalid JSON: %v\n", err)
				os.Exit(1)
			}
		}

		if payload.Id == "" {
			fmt.Fprintln(os.Stderr, "[ERROR] Missing required field: \"id\"")
			os.Exit(1)
		}
		if payload.Command == "" {
			fmt.Fprintln(os.Stderr, "[ERROR] Missing required field: \"command\"")
			os.Exit(1)
		}

		var runAtVal interface{} = nil
		if payload.RunAt != "" {
			_, err := time.Parse("2006-01-02T15:04:05Z", payload.RunAt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Invalid run_at timestamp format. Use ISO8601 UTC format: YYYY-MM-DDTHH:MM:SSZ\n")
				os.Exit(1)
			}
			runAtVal = payload.RunAt
		}

		timeoutVal := payload.Timeout
		if timeoutVal <= 0 {
			timeoutVal = 600 // Default to 10 minutes if not specified
		}

		maxRetries := getMaxRetries()

		query := `
			INSERT INTO jobs (id, command, state, attempts, max_retries, priority, run_at, timeout)
			VALUES (?, ?, 'pending', 0, ?, ?, ?, ?)
		`

		_, err := db.DB.Exec(query, payload.Id, payload.Command, maxRetries, payload.Priority, runAtVal, timeoutVal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to enqueue job: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Job enqueued successfully!\n")
		fmt.Printf("   ID:       %s\n", payload.Id)
		fmt.Printf("   Command:  %s\n", payload.Command)
		fmt.Printf("   State:    pending\n")
		fmt.Printf("   Priority: %d\n", payload.Priority)
		if payload.RunAt != "" {
			fmt.Printf("   Run At:   %s\n", payload.RunAt)
		}
		fmt.Printf("   Timeout:  %ds\n", timeoutVal)
		fmt.Printf("   Retries:  0/%d\n", maxRetries)
	},
}

func init() {
	enqueueCmd.Flags().StringVar(&flagJobID, "id", "", "ID of the job to enqueue")
	enqueueCmd.Flags().StringVar(&flagCommand, "command", "", "Shell command for the job to execute")
	enqueueCmd.Flags().IntVar(&flagPriority, "priority", 0, "Priority level (higher values run first)")
	enqueueCmd.Flags().StringVar(&flagRunAt, "run-at", "", "ISO8601 UTC timestamp to schedule job execution")
	enqueueCmd.Flags().IntVar(&flagTimeout, "timeout", 600, "Maximum execution time in seconds")
}

func getMaxRetries() int {
	var value int
	err := db.DB.QueryRow("SELECT value FROM config WHERE key = 'max_retries'").Scan(&value)
	if err != nil {
		return 3
	}
	return value
}
