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

var enqueueCmd = &cobra.Command{
	Use:   "enqueue [JSON payload]",
	Short: "Add a new job to the queue",
	Long: `Enqueue adds a new job to the processing queue.

The payload must be a valid JSON string with "id" and "command" fields.

Example:
  queuectl enqueue '{"id":"job1", "command":"echo hello world"}'
  queuectl enqueue '{"id":"job2", "command":"sleep 5"}'`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// args[0] is the raw JSON string from the terminal
		jsonInput := args[0]

		// --- Step 1: Parse JSON into a Go struct ---
		var payload JobPayload

		// json.Unmarshal converts a JSON string (as bytes) into a struct.
		// If the JSON is malformed, it returns an error.
		if err := json.Unmarshal([]byte(jsonInput), &payload); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Invalid JSON: %v\n", err)
			os.Exit(1)
		}

		// --- Step 2: Validate required fields ---
		if payload.Id == "" {
			fmt.Fprintln(os.Stderr, "❌ Missing required field: \"id\"")
			os.Exit(1)
		}
		if payload.Command == "" {
			fmt.Fprintln(os.Stderr, "❌ Missing required field: \"command\"")
			os.Exit(1)
		}

		// --- Step 3: Read current config for max_retries ---
		maxRetries := getMaxRetries()

		// --- Step 4: Insert into the database ---
		// The "?" placeholders prevent SQL injection attacks.
		// Go fills them in order: $1=id, $2=command, $3=max_retries
		query := `
			INSERT INTO jobs (id, command, state, attempts, max_retries)
			VALUES (?, ?, 'pending', 0, ?)
		`

		_, err := db.DB.Exec(query, payload.Id, payload.Command, maxRetries)
		if err != nil {
			// If the id already exists, SQLite will throw a UNIQUE constraint error
			fmt.Fprintf(os.Stderr, "❌ Failed to enqueue job: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Job enqueued successfully!\n")
		fmt.Printf("   ID:      %s\n", payload.Id)
		fmt.Printf("   Command: %s\n", payload.Command)
		fmt.Printf("   State:   pending\n")
		fmt.Printf("   Retries: 0/%d\n", maxRetries)
	},
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
