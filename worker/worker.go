package worker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
)

func StartWorkers(count int) {
	fmt.Printf("[INFO] Starting %d worker(s)...\n", count)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID)
		}(i)
	}

	fmt.Printf("[INFO] All %d worker(s) running. Press Ctrl+C to stop gracefully.\n", count)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\n[INFO] Shutdown signal received. Waiting for workers to finish current jobs...")
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
