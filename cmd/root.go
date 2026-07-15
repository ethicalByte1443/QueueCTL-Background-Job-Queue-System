/*
=============================================================================
🎓 LEARNING NOTE — cmd/root.go (Root Command)
=============================================================================

WHAT THIS FILE DOES:
  This is the BASE command for our CLI. When someone types just `queuectl`
  with no subcommand, THIS is what runs. All other commands (enqueue, worker,
  status, etc.) are "children" of this root command.

KEY GO CONCEPTS:

  1. COBRA COMMANDS
     Cobra works with a tree of commands:
       queuectl              ← ROOT (this file)
       ├── enqueue           ← child command
       ├── worker            ← child command
       │   ├── start         ← grandchild command
       │   └── stop          ← grandchild command
       ├── status            ← child command
       ├── list              ← child command
       ├── dlq               ← child command
       │   ├── list          ← grandchild command
       │   └── retry         ← grandchild command
       └── config            ← child command

  2. &cobra.Command{} — STRUCT LITERAL
     In Go, you create objects by filling in a "struct" (like a class with
     only fields, no methods). The & means "give me a pointer to this struct"
     so we can pass it around efficiently.

  3. init() FUNCTION
     Go has a special function called init() that runs automatically when the
     package is loaded — before main() is called. We use it here to register
     all subcommands to the root command. You never call init() yourself.

=============================================================================
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command. When the user types just "qcli",
// Cobra runs the function in the "Run" field.
var rootCmd = &cobra.Command{
	Use:   "qcli",
	Short: "A CLI-based background job queue system",
	Long: `QCli is a background job queue system that manages tasks with
worker processes, automatic retries using exponential backoff,
and a Dead Letter Queue (DLQ) for permanently failed jobs.

Usage examples:
  qcli enqueue '{"id":"job1", "command":"echo hello"}'
  qcli worker start --count 3
  qcli status
  qcli dlq list`,
}

// Execute is called from main.go. It starts Cobra's command parsing.
// If the user types an unknown command, Cobra will print an error and
// suggest similar commands automatically.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// init() runs automatically. We register all child commands here.
func init() {
	// Each of these AddCommand calls attaches a subcommand to the root.
	// The actual command definitions are in their own files (enqueue.go, etc.)
	rootCmd.AddCommand(enqueueCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(dlqCmd)
	rootCmd.AddCommand(configCmd)
}
