/*
=============================================================================
🎓 LEARNING NOTE — main.go (Application Entry Point)
=============================================================================

WHAT THIS FILE DOES:
  This is the STARTING POINT of our entire application. When you run
  `queuectl` in the terminal, Go looks for a file called main.go with a
  function called main() — that's where execution begins.

KEY GO CONCEPTS:

  1. package main
     The special package name "main" tells Go: "this is an executable program,
     not a library." Every Go app that produces a binary must have exactly one
     package main with one func main().

  2. func main()
     This is the entry point — like if __name__ == "__main__" in Python.
     We call cmd.Execute() here which hands control over to Cobra (our CLI
     framework). Cobra then figures out which subcommand the user typed
     (enqueue, worker, status, etc.) and calls the right function.

  3. defer db.Close()
     Remember "defer" from db.go? Here we use it to guarantee the database
     connection is closed when the program exits, no matter how it exits
     (success, error, crash). It's like a "finally" block in other languages.

=============================================================================
*/

package main

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/cmd"
	"github.com/ethicalByte1443/queuectl/db"
)

func main() {
	// Step 1: Initialize the database connection.
	// If this fails (e.g., disk full, permissions), we print the error and exit.
	if err := db.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize database: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Schedule database cleanup when the program exits.
	defer db.Close()

	// Step 3: Hand control to Cobra to parse the user's command.
	cmd.Execute()
}
