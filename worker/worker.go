package worker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/ethicalByte1443/queuectl/db"
)

func StartWorkers(count int) {
	fmt.Printf("[INFO] Starting %d worker(s)...\n", count)

	pid := os.Getpid()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Register worker process in database
	_, err := db.DB.Exec(
		`INSERT INTO worker_processes (pid, worker_count, status, started_at, updated_at)
		 VALUES (?, ?, 'running', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(pid) DO UPDATE SET worker_count = excluded.worker_count, status = 'running', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		pid, count,
	)
	if err != nil {
		fmt.Printf("[WARNING] Failed to register worker process in database: %v\n", err)
	}

	defer func() {
		// Clean up worker process registration on exit
		_, _ = db.DB.Exec("DELETE FROM worker_processes WHERE pid = ?", pid)
	}()

	// Start worker goroutines
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID)
		}(i)
	}

	fmt.Printf("[INFO] All %d worker(s) running. Press Ctrl+C or run 'queuectl worker stop' to stop gracefully.\n", count)

	// Heartbeat & stop signal checker
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Update heartbeat
				_, err := db.DB.Exec(
					"UPDATE worker_processes SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE pid = ? AND status = 'running'",
					pid,
				)
				if err != nil {
					// Ignore transient db lock/connection errors during heartbeat
				}

				// Check if stop is requested for this process
				var status string
				err = db.DB.QueryRow("SELECT status FROM worker_processes WHERE pid = ?", pid).Scan(&status)
				if err == nil && status == "stop_requested" {
					fmt.Printf("\n[INFO] Remote stop signal received for worker process %d. Initiating graceful shutdown...\n", pid)
					cancel()
					return
				}
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	select {
	case <-sigChan:
		fmt.Println("\n[INFO] Shutdown signal (Ctrl+C) received. Waiting for workers to finish current jobs...")
	case <-ctx.Done():
		// Triggered by database stop signal
	}

	cancel()
	wg.Wait()

	fmt.Println("[INFO] All workers stopped gracefully.")
}

func runWorker(ctx context.Context, workerID int) {
	fmt.Printf("  [Worker %d] Started\n", workerID)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("  [Worker %d] Shutting down...\n", workerID)
			return
		default:
		}

		jobFound := claimAndRunJob(ctx, workerID)
		if !jobFound {
			select {
			case <-ctx.Done():
				fmt.Printf("  [Worker %d] Shutting down...\n", workerID)
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}
