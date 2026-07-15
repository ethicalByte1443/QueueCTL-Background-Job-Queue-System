/*
=============================================================================
🎓 LEARNING NOTE — cmd/enqueue.go (Enqueue Subcommand)
=============================================================================

WHAT THIS FILE DOES:
  Defines the "enqueue" subcommand. When the user types:
    queuectl enqueue '{"id":"job1", "command":"echo hello"}'
  Cobra calls the Run function here.

KEY GO CONCEPT — cobra.Command Args:

  cobra.ExactArgs(1) tells Cobra: "this command requires EXACTLY 1 argument."
  If the user gives 0 or 2+ arguments, Cobra automatically prints an error.
  The argument (the JSON string) is available as args[0].

  For now, this is a skeleton — Task 4 will fill in the real logic.

=============================================================================
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var enqueueCmd = &cobra.Command{
	Use:   "enqueue [JSON payload]",
	Short: "Add a new job to the queue",
	Long: `Enqueue adds a new job to the processing queue.

The payload must be a valid JSON string with "id" and "command" fields.

Example:
  queuectl enqueue '{"id":"job1", "command":"echo hello world"}'
  queuectl enqueue '{"id":"job2", "command":"sleep 5"}'`,
	Args: cobra.ExactArgs(1), // Require exactly 1 argument (the JSON string)
	Run: func(cmd *cobra.Command, args []string) {
		// args[0] contains the JSON payload string
		// TODO: Task 4 will implement JSON parsing and DB insertion here
		fmt.Printf("📥 Enqueue called with payload: %s\n", args[0])
	},
}
