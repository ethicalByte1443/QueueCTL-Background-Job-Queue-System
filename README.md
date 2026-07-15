# QueueCTL - Background Job Queue System

QueueCTL is a production-grade, highly concurrent CLI-based background job queue system written in Go, powered by SQLite. It manages background tasks, handles automatic retries with exponential backoff, runs parallel worker processes, maintains a Dead Letter Queue (DLQ), and hosts a visual web dashboard.

---

## 1. Setup Instructions

### Prerequisites
- Install **Go (v1.18+)** from [go.dev](https://go.dev/).

### Compilation
Clone the repository, open your terminal in the workspace root, and run:
```bash
# Compile the production-grade binary
go build -o queuectl.exe .
```

### Initializing the System
When you run any `queuectl` command, it will automatically initialize a persistent SQLite database located at `~/.queuectl/queuectl.db` and create the required schema tables.

---

## 2. Usage Examples

QueueCTL supports the following commands:

### `config` & `config set`
View or modify maximum retries and backoff base intervals:
```bash
# 1. View current configuration
.\queuectl.exe config
```
**Example Output:**
```text
Current Configuration:
  Max Retries:  3
  Backoff Base: 2 seconds
```

```bash
# 2. Update config parameters using flags
.\queuectl.exe config --max-retries 5 --backoff-base 3
```
**Example Output:**
```text
[SUCCESS] max-retries set to 5
[SUCCESS] backoff-base set to 3 seconds
```

```bash
# 3. Update config parameters using positional command set
.\queuectl.exe config set max-retries 3
```
**Example Output:**
```text
[SUCCESS] Config parameter 'max_retries' updated to 3
```

---

### `enqueue`
Add new tasks to the queue using command-line flags or raw JSON payloads. Supports priority, scheduling execution (`run-at`), and timeout execution limits:
```bash
# 1. Enqueue job using flags
.\queuectl.exe enqueue --id job1 --command "echo 'hello world'" --priority 10 --timeout 30
```
**Example Output:**
```text
[DB] Initialized at: C:\Users\Aseem\.queuectl\queuectl.db
Job enqueued successfully!
   ID:       job1
   Command:  echo 'hello world'
   State:    pending
   Priority: 10
   Timeout:  30s
   Retries:  0/3
```

```bash
# 2. Enqueue job using JSON string payload
.\queuectl.exe enqueue '{"id":"job2","command":"sleep 2","priority":5,"run_at":"2026-07-16T12:00:00Z"}'
```
**Example Output:**
```text
Job enqueued successfully!
   ID:       job2
   Command:  sleep 2
   State:    pending
   Priority: 5
   Run At:   2026-07-16T12:00:00Z
   Timeout:  600s
   Retries:  0/3
```

---

### `worker`
Manage background execution worker daemons:
```bash
# 1. Start parallel worker goroutines in the foreground
.\queuectl.exe worker start --count 3
```
**Example Output:**
```text
[INFO] Starting 3 worker(s)...
[INFO] All 3 worker(s) running. Press Ctrl+C or run 'queuectl worker stop' to stop gracefully.
  [Worker 1] Started
  [Worker 2] Started
  [Worker 3] Started
  [Worker 1] [INFO] Claimed job: job1 (command: echo 'hello world', attempt: 1/3)
  [Worker 1] [SUCCESS] Job job1 completed successfully
```

```bash
# 2. Stop running worker processes gracefully from another terminal
.\queuectl.exe worker stop
```
**Example Output:**
```text
[INFO] Sent graceful stop signal to 1 running worker process(es).
       Workers will finish their current jobs and stop gracefully.
```

---

### `status`, `list`, `stats`, & `logs`
Monitor and analyze jobs in the queue:
```bash
# 1. Show overall queue counts and active worker processes
.\queuectl.exe status
```
**Example Output:**
```text
Queue Status
  ────────────────────────────────────────
  Pending:       0
  Processing:    0
  Completed:     1
  Failed:        0
  Dead (DLQ):    1
  ────────────────────────────────────────
  Total Jobs:    2

Active Workers
  ────────────────────────────────────────
  Worker Processes:  1
  Total Goroutines:  3

  Running Processes:
    PID      GOROUTINES   STARTED AT               
    ────────────────────────────────────────────
    7828     3            2026-07-15T20:55:24Z     
```

```bash
# 2. List jobs in the queue, optionally filtering by state
.\queuectl.exe list --state pending
```
**Example Output:**
```text
ID             COMMAND                   STATE        ATTEMPTS   ERROR
──────────────────────────────────────────────────────────────────────────────────────────
job2           sleep 2                   pending      0/3        -

Total: 1 job(s)
```

```bash
# 3. View telemetry and execution statistics
.\queuectl.exe stats
```
**Example Output:**
```text
Queue CTL - System Performance Metrics
==============================================================
Job Counts Summary:
  Total Enqueued Jobs:   2
  Pending Jobs:          1
  Processing Jobs:       0
  Completed Jobs:        1
  Failed (Retryable):    0
  Dead Letter (DLQ):     1
--------------------------------------------------------------
Efficiency & Reliability:
  Success Rate:          50.00%
  Avg Execution Time:    48.00 ms
  Min Execution Time:    48.00 ms
  Max Execution Time:    48.00 ms
--------------------------------------------------------------
Active Infrastructure:
  Worker Processes:      1
  Worker Goroutines:     3
==============================================================
```

```bash
# 4. View stdout/stderr and error logs for a specific job
.\queuectl.exe logs job-fail
```
**Example Output:**
```text
==============================================================
Job Details: job-fail
==============================================================
  Command:    non_existent_command
  State:      dead
  Attempts:   3/3
  Duration:   17 ms
  Created At: 2026-07-15T20:55:03Z
  Updated At: 2026-07-15T20:55:30Z
==============================================================
Error Message:
exit status 1
==============================================================
Execution Output:
(No output was captured)
==============================================================
```

---

### `dlq`
Inspect and retry jobs that have permanently failed:
```bash
# 1. List dead jobs in Dead Letter Queue (DLQ)
.\queuectl.exe dlq list
```
**Example Output:**
```text
Dead Letter Queue
ID             COMMAND                   ATTEMPTS   ERROR                                   
──────────────────────────────────────────────────────────────────────────────────────────
job-fail       non_existent_command      3/3        exit status 1                           

Total: 1 dead job(s)
```

```bash
# 2. Re-queue dead job back to pending status
.\queuectl.exe dlq retry job-fail
```
**Example Output:**
```text
[INFO] Job 'job-fail' moved back to pending. It will be picked up by workers.
```

---

### `dashboard`
Start the visual web-based monitoring server:
```bash
# Start the web server (defaults to port 8080)
.\queuectl.exe dashboard --port 8080
```
**Example Output:**
```text
[INFO] QueueCTL Dashboard starting on http://127.0.0.1:8080
```

---

## 3. Architecture Overview

### Job Life Cycle
QueueCTL coordinates jobs through 5 distinct states:
```
                  [ enqueue ]
                       │
                       ▼
                 [ pending ] ──(Worker Claim)──> [ processing ]
                      ▲                                 │
                      │ (Backoff delay elapsed)         ├──(Success)──> [ completed ]
                      └─────── [ failed ] ◄─────────────┤
                                                        └──(Retries exhausted)──> [ dead ] (DLQ)
```

### Data Persistence
SQLite is utilized as the persistent central coordinator. It stores configuration settings, records active daemon registrations, and hosts the complete list of jobs.
- The SQLite database uses **Write-Ahead Logging (WAL)** mode and a `busy_timeout` config of 5000ms. This prevents read/write blockage and enables SQLite to handle massive concurrent operations.

### Worker Logic
1. **Polling Loop**: Goroutines continuously poll the database looking for ready jobs (state is `pending`, or state is `failed` and backoff delay has elapsed).
2. **Atomic Locking & Claiming**: To prevent duplicate processing under heavy concurrent thread loads, workers use a serialized update check. They execute:
   ```sql
   UPDATE jobs SET state = 'processing' WHERE id = ? AND state = ?
   ```
   Only the thread that successfully updates the row (returns `RowsAffected() == 1`) executes the command.
3. **Execution & Metrics**: The command runs in a separate child shell process. The execution time is tracked, and output/exit status is saved to the database.
4. **Exponential Backoff**: Delay calculation is done using the formula:
   $$\text{delay} = \text{backoff\_base}^{\text{attempts}}\text{ seconds}$$
   If a job reaches `max_retries`, it transitions into `dead` state (DLQ).

---

## 4. Assumptions & Trade-offs

### 1. Database Polling vs. Event Hooks
- **Decision**: SQLite does not support standard event triggers/pub-sub notifications to notify external processes when a job is enqueued. Therefore, worker threads poll the database on a interval ticker (defaulting to 2-second sleep if no jobs are found).
- **Trade-off**: This increases database read frequency slightly but remains extremely lightweight because SQLite is embedded directly in-process.

### 2. Database-Backed Signals for Graceful Stop
- **Decision**: Unix signal handlers (like `SIGUSR1`) behave inconsistently on Windows. We implemented worker process stop signaling by updating a status flag inside the `worker_processes` table in SQLite.
- **Trade-off**: This guarantees stop commands work reliably across platforms (including Windows, macOS, and Linux) without custom native OS bindings.

---

## 5. Testing Instructions

### Automated Unit Tests
We have built unit and integration tests to verify claims, retries, and priority scheduling. Run:
```bash
go test -v ./worker
```

### Manual Integration Scenario
To verify that core features behave correctly under real execution conditions:

1. **Verify Successful Execution**:
   ```bash
   .\queuectl.exe enqueue --id job-ok --command "echo 'success'"
   .\queuectl.exe worker start --count 1
   ```
   *Observe the worker runs `echo 'success'`, updates the state to `completed`, and prints SUCCESS.*

2. **Verify Failures & Backoff Retries**:
   ```bash
   .\queuectl.exe enqueue --id job-fail --command "invalid_command_xyz"
   ```
   *Observe the worker logs the failure, increments the attempts counter, schedules retry backoffs, and finally moves the job to the DLQ after 3 failures.*

3. **Verify Concurrency & Overlap Prevention**:
   Start two separate worker processes from two terminals:
   ```bash
   # Terminal A
   .\queuectl.exe worker start --count 2
   
   # Terminal B
   .\queuectl.exe worker start --count 2
   ```
   Enqueue multiple sleep jobs and verify they are processed in parallel without overlapping/duplicate execution.