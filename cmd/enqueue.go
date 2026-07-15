package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

type JobPayload struct {
	Id      string `json:"id"`
	Command string `json:"command"`
}

var (
	flagJobID   string
	flagCommand string
)

var enqueueCmd = &cobra.Command{
	Use:   "enqueue [JSON payload]",
	Short: "Add a new job to the queue",
	Long: `Enqueue adds a new job to the processing queue.

You can either pass a valid JSON string with "id" and "command" fields,
or use the --id and --command flags directly.

Examples:
  qcli enqueue --id job1 --command "echo hello world"
  qcli enqueue '{"id":"job1", "command":"echo hello world"}'`,
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

		maxRetries := getMaxRetries()

		query := `
			INSERT INTO jobs (id, command, state, attempts, max_retries)
			VALUES (?, ?, 'pending', 0, ?)
		`

		_, err := db.DB.Exec(query, payload.Id, payload.Command, maxRetries)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to enqueue job: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Job enqueued successfully!\n")
		fmt.Printf("   ID:      %s\n", payload.Id)
		fmt.Printf("   Command: %s\n", payload.Command)
		fmt.Printf("   State:   pending\n")
		fmt.Printf("   Retries: 0/%d\n", maxRetries)
	},
}

func init() {
	enqueueCmd.Flags().StringVar(&flagJobID, "id", "", "ID of the job to enqueue")
	enqueueCmd.Flags().StringVar(&flagCommand, "command", "", "Shell command for the job to execute")
}

func getMaxRetries() int {
	var value int
	err := db.DB.QueryRow("SELECT value FROM config WHERE key = 'max_retries'").Scan(&value)
	if err != nil {
		return 3
	}
	return value
}
