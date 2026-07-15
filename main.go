package main

import (
	"fmt"
	"os"

	"github.com/ethicalByte1443/queuectl/cmd"
	"github.com/ethicalByte1443/queuectl/db"
)

func main() {
	if err := db.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	cmd.Execute()
}
