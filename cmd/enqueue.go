/*
=============================================================================
🎓 LEARNING NOTE — cmd/enqueue.go (Full Implementation)
=============================================================================

WHAT THIS FILE DOES:
  Now implements the REAL enqueue logic:
  1. Parse the JSON payload from the user
  2. Validate it has "id" and "command" fields
  3. Insert the job into SQLite with state = "pending"

KEY GO CONCEPTS:

  1. STRUCTS — type JobPayload struct { ... }
     A struct is Go's version of a "class" (but simpler — no inheritance).
     It's a collection of named fields. We define one to hold the parsed
     JSON data.

  2. JSON TAGS — `json:"id"`
     The backtick part after each field tells Go's JSON parser which JSON
     key maps to which struct field. So {"id":"job1"} maps to the Id field.

  3. json.Unmarshal([]byte(input), &payload)
     "Unmarshal" means "convert JSON text → Go struct". The opposite
     (struct → JSON) is called "Marshal". Think of it as:
       Unmarshal = JSON string → Go object (deserialize)
       Marshal   = Go object → JSON string (serialize)

  4. PREPARED STATEMENTS — db.DB.Exec(query, args...)
     Instead of concatenating SQL strings (which is dangerous — SQL injection!),
     we use "?" placeholders. Go fills them in safely.

=============================================================================
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

// JobPayload represents the JSON structure the user provides.
// The `json:"id"` tag tells Go: when you see "id" in JSON, put it in Id.
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
or use the user-friendly --id and --command flags directly.

Examples:
  # Using flags (Recommended for Windows):
  qcli enqueue --id job1 --command "echo hello world"

  # Using raw JSON payload:
  qcli enqueue '{"id":"job1", "command":"echo hello world"}'`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var payload JobPayload

		// Check if flags are provided
		if flagJobID != "" || flagCommand != "" {
			if flagJobID == "" || flagCommand == "" {
				fmt.Fprintln(os.Stderr, "[ERROR] Both --id and --command flags must be provided if using flags.")
				os.Exit(1)
			}
			payload.Id = flagJobID
			payload.Command = flagCommand
		} else {
			// Fallback to JSON payload argument
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

		// --- Step 2: Validate required fields ---
		if payload.Id == "" {
			fmt.Fprintln(os.Stderr, "[ERROR] Missing required field: \"id\"")
			os.Exit(1)
		}
		if payload.Command == "" {
			fmt.Fprintln(os.Stderr, "[ERROR] Missing required field: \"command\"")
			os.Exit(1)
		}

		// --- Step 3: Read current config for max_retries ---
		maxRetries := getMaxRetries()

		// --- Step 4: Insert into the database ---
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

// getMaxRetries reads the max_retries config from the config table.
// Falls back to the default value of 3 if not set.
func getMaxRetries() int {
	var value int
	err := db.DB.QueryRow("SELECT value FROM config WHERE key = 'max_retries'").Scan(&value)
	if err != nil {
		return 3 // default
	}
	return value
}
