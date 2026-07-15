# QCli

A CLI-based background job queue system written in Go.

## Setup

Build the executable binary:
```bash
go build -o qcli.exe .
```

## CLI Commands

### Configuration
Manage maximum retries and backoff base intervals:
```bash
# View configuration
.\qcli.exe config

# Set max retries and backoff base
.\qcli.exe config --max-retries 5 --backoff-base 3
```

### Enqueue Jobs
Add tasks using flags (Windows) or raw JSON payloads:
```bash
# Using flags
.\qcli.exe enqueue --id job1 --command "echo hello"

# Using JSON payload
.\qcli.exe --% enqueue "{\"id\":\"job1\",\"command\":\"echo hello\"}"
```

### Status & List
Monitor the queue status and list tasks:
```bash
# View summary stats
.\qcli.exe status

# List all tasks
.\qcli.exe list

# Filter list by state
.\qcli.exe list --state pending
```

### Worker Management
Start worker processes:
```bash
# Start 1 worker
.\qcli.exe worker start

# Start parallel workers
.\qcli.exe worker start --count 3
```
*Stop workers by pressing `Ctrl+C`.*

### Dead Letter Queue (DLQ)
Inspect and retry permanently failed tasks:
```bash
# List dead tasks
.\qcli.exe dlq list

# Re-queue dead task
.\qcli.exe dlq retry job1
```

## Architecture & Flow

The system runs as a lightweight, concurrent task orchestrator:
1. **Producer:** `qcli enqueue` inserts task payloads directly into the database.
2. **Database:** SQLite stores task configurations, run statuses, execution metrics, and logs.
3. **Consumers:** Parallel workers run inside Go goroutines. They coordinate job picking, subprocess execution, error capturing, and retries.

### Job Life Cycle
```
[ pending ] ──(Worker Claim)──> [ processing ]
     ▲                                 │
     │ (Backoff delay elapsed)         ├──(Success)──> [ completed ]
     └─────── [ failed ] ◄─────────────┤
                                       └──(Retries exhausted)──> [ dead ] (DLQ)
```

## Challenges & Solutions

### 1. SQLite Database Concurrency
* **Problem:** Multiple parallel workers updating task states concurrently cause database lock conflicts (`SQLITE_BUSY` errors).
* **Solution:** 
  * Configured connection params (`busy_timeout=5000`) so workers wait for locks to clear instead of throwing errors.
  * Wrapped lock-and-claim operations inside a transaction block using `BEGIN IMMEDIATE` to lock the DB during the pick phase.

### 2. Missing SQLite Math Functions
* **Problem:** Calculating exponential backoff delay intervals (`base^attempts`) directly inside the database query requires math capabilities not natively available in SQLite without extensions.
* **Solution:** Simplified the database query to fetch candidate failed jobs, then evaluated backoff eligibility inside Go code using standard library `math.Pow`.

## License
MIT License