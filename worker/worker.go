/*
=============================================================================
🎓 LEARNING NOTE — worker/worker.go (Worker Pool Coordinator)
=============================================================================

WHAT THIS FILE DOES:
  This file manages the "worker pool" — it starts multiple workers in
  parallel and coordinates their lifecycle (start, run, graceful shutdown).

KEY GO CONCEPTS:

  1. GOROUTINES — go functionName()
     A goroutine is a LIGHTWEIGHT THREAD. When you write "go doWork()",
     Go starts doWork() in the background and immediately moves to the
     next line. It's like hiring a new employee — they work independently.

     Unlike regular threads (which use ~1MB of RAM each), goroutines use
     only ~2KB. So you can easily run thousands of them.

  2. sync.WaitGroup
     A WaitGroup is a COUNTER that tracks how many goroutines are running.
       wg.Add(1)  → "one more worker started"
       wg.Done()  → "one worker finished"
       wg.Wait()  → "block here until ALL workers are done"
     This ensures we don't exit the program while workers are still busy.

  3. context.Context — CANCELLATION SIGNAL
     A context is like a "cancel button" that you pass to all goroutines.
     When someone calls cancel(), EVERY goroutine watching ctx.Done() gets
     notified. It's Go's way of saying "everyone stop what you're doing."

     ctx, cancel := context.WithCancel(context.Background())
     - ctx     → pass this to workers; they watch ctx.Done()
     - cancel  → call this to send the stop signal

  4. os/signal.Notify — CATCHING Ctrl+C
     When you press Ctrl+C in the terminal, the OS sends a SIGINT signal.
     signal.Notify(sigChan, os.Interrupt) captures this signal into a
     Go channel, so we can handle it gracefully instead of crashing.

  5. CHANNELS — make(chan os.Signal, 1)
     A channel is a PIPE between goroutines. One goroutine puts data in,
     another takes data out. It's like a mailbox:
       sigChan <- signal  → put a letter in the mailbox
       <-sigChan          → wait and take a letter out

     The "1" in make(chan os.Signal, 1) is the buffer size — it can hold
     1 signal without blocking the sender.

=============================================================================
*/

package worker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
)

// StartWorkers launches 'count' workers and blocks until they all finish.
// Workers run until a SIGINT (Ctrl+C) signal is received.
func StartWorkers(count int) {
	fmt.Printf("[INFO] Starting %d worker(s)...\n", count)

	// --- Step 1: Create a cancellation context ---
	// Think of this as a "kill switch" shared by all workers.
	ctx, cancel := context.WithCancel(context.Background())

	// --- Step 2: Create a WaitGroup to track active workers ---
	var wg sync.WaitGroup

	// --- Step 3: Launch workers as goroutines ---
	for i := 1; i <= count; i++ {
		wg.Add(1) // Tell the WaitGroup: "one more worker"

		// "go" keyword starts this function in a separate goroutine.
		// Each worker gets its own ID number (i) for logging.
		go func(workerID int) {
			defer wg.Done() // When this function exits, tell WaitGroup: "one less worker"
			runWorker(ctx, workerID)
		}(i)
	}

	fmt.Printf("[INFO] All %d worker(s) running. Press Ctrl+C to stop gracefully.\n", count)

	// --- Step 4: Wait for Ctrl+C signal ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// This line BLOCKS until Ctrl+C is pressed.
	// The program just sits here, while workers run in the background.
	<-sigChan

	fmt.Println("\n[INFO] Shutdown signal received. Waiting for workers to finish current jobs...")

	// --- Step 5: Tell all workers to stop ---
	cancel() // This triggers ctx.Done() in all workers

	// --- Step 6: Wait for all workers to finish their current jobs ---
	wg.Wait()

	fmt.Println("[INFO] All workers stopped gracefully.")
}

// runWorker is the main loop for a single worker.
// It continuously polls the database for pending jobs and executes them.
func runWorker(ctx context.Context, workerID int) {
	fmt.Printf("  [Worker %d] Started\n", workerID)

	for {
		// Check if we've been told to stop
		select {
		case <-ctx.Done():
			// The cancel() function was called — time to stop.
			fmt.Printf("  [Worker %d] Shutting down...\n", workerID)
			return
		default:
			// Not cancelled yet — keep working
		}

		// Try to claim and execute a job from the database.
		// claimAndRunJob is defined in job_runner.go
		jobFound := claimAndRunJob(ctx, workerID)

		if !jobFound {
			// No jobs available right now. Wait a bit before checking again.
			// This prevents the worker from hammering the database in a tight loop.
			select {
			case <-ctx.Done():
				fmt.Printf("  [Worker %d] Shutting down...\n", workerID)
				return
			case <-time.After(2 * time.Second):
				// Wait 2 seconds, then check for jobs again
			}
		}
	}
}
