package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
			var err error
			payload, err = parseFlexiblePayload(jsonInput)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Invalid payload: %v\n", err)
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

func parseFlexiblePayload(input string) (JobPayload, error) {
	input = strings.Trim(input, "'")
	input = strings.TrimSpace(input)

	var payload JobPayload

	// Try standard JSON parsing first
	err := json.Unmarshal([]byte(input), &payload)
	if err == nil {
		return payload, nil
	}

	// Keys to search for
	keys := []string{"id:", "command:", "priority:", "run_at:", "timeout:"}
	
	type keyPos struct {
		key      string
		valStart int
	}
	var positions []keyPos

	for _, k := range keys {
		idx := strings.Index(input, k)
		if idx != -1 {
			positions = append(positions, keyPos{key: k, valStart: idx + len(k)})
		}
	}

	// Sort positions by valStart
	for i := 0; i < len(positions); i++ {
		for j := i + 1; j < len(positions); j++ {
			if positions[i].valStart > positions[j].valStart {
				positions[i], positions[j] = positions[j], positions[i]
			}
		}
	}

	if len(positions) == 0 {
		return payload, fmt.Errorf("no valid keys found in payload: %s", input)
	}

	// Extract values
	for i, pos := range positions {
		endIdx := len(input)
		if i+1 < len(positions) {
			nextKeyStart := positions[i+1].valStart - len(positions[i+1].key)
			endIdx = nextKeyStart
		}

		val := input[pos.valStart:endIdx]
		
		val = strings.TrimSpace(val)
		val = strings.TrimSuffix(val, ",")
		val = strings.TrimSuffix(val, "}")
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)

		switch pos.key {
		case "id:":
			payload.Id = val
		case "command:":
			payload.Command = val
		case "priority:":
			fmt.Sscanf(val, "%d", &payload.Priority)
		case "run_at:":
			payload.RunAt = val
		case "timeout:":
			fmt.Sscanf(val, "%d", &payload.Timeout)
		}
	}

	return payload, nil
}
