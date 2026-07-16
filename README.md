# QueueCTL

QueueCTL is a lightweight, concurrent CLI background job queue system written in Go and backed by SQLite. 

---

## Setup Instructions

### Prerequisites
- **Go 1.18+** 

### Build
To compile the QueueCTL, run:
```bash
go build -o queuectl.exe .
```

### Database Initialization
The application initializes automatically on the first command run. It creates a local SQLite database at `~/.queuectl/queuectl.db` (under your user home directory) and sets up the required tables for job queues, worker telemetry, and metrics.

---

## Usage Examples

Below are the commands supported by the CLI, along with realistic example outputs.

### Configuration
View the current runtime configurations (retries and backoff limits) or modify them.

#### View Config
```bash
.\queuectl.exe config
```
**Output:**
```text
Current Configuration:
  Max Retries:  3
  Backoff Base: 2 seconds
```

#### Set Config via Flags
```bash
.\queuectl.exe config --max-retries 5 --backoff-base 3
```
**Output:**
```text
[SUCCESS] max-retries set to 5
[SUCCESS] backoff-base set to 3 seconds
```

#### Set Config via Positional Subcommand
```bash
.\queuectl.exe config set max-retries 3
```
**Output:**
```text
[SUCCESS] Config parameter 'max_retries' updated to 3
```

---

### Enqueueing Jobs
Add new tasks to the queue using command-line flags or a raw JSON payload. The system supports job priority, scheduled execution delays (`run-at`), and timeout limits. 

*Note: The CLI natively parses quote-stripped JSON arguments typed directly into PowerShell, so no manual backslash escaping is required.*

#### Option A: Enqueue using CLI flags
```bash
.\queuectl.exe enqueue --id job1 --command "echo 'hello world'" --priority 10 --timeout 30
```
**Output:**
```text
Job enqueued successfully!
   ID:       job1
   Command:  echo 'hello world'
   State:    pending
   Priority: 10
   Timeout:  30s
   Retries:  0/3
```

#### Option B: Enqueue using a JSON payload string
```bash
.\queuectl.exe enqueue '{"id":"job2","command":"sleep 2","priority":5,"run_at":"2026-07-16T12:00:00Z"}'
```
**Output:**
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

### Managing Workers
Workers run in parallel goroutines. You can control daemon instances using start/stop operations.

#### Start Worker Goroutines
```bash
.\queuectl.exe worker start --count 2
```
**Output:**
```text
[INFO] Starting 2 worker(s)...
[INFO] All 2 worker(s) running. Press Ctrl+C or run 'queuectl worker stop' to stop gracefully.
  [Worker 1] Started
  [Worker 2] Started
  [Worker 1] [INFO] Claimed job: job1 (command: echo 'hello world', attempt: 1/3)
  [Worker 1] [SUCCESS] Job job1 completed successfully
```

#### Gracefully Stop Workers (Remote Signal)
```bash
.\queuectl.exe worker stop
```
**Output:**
```text
[INFO] Sent graceful stop signal to 1 running worker process(es).
       Workers will finish their current jobs and stop gracefully.
```

---

### Queue Inspection & Logs

#### Show Current Status
Lists aggregate counts and registered worker process IDs (PIDs):
```bash
.\queuectl.exe status
```
**Output:**
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
  Total Goroutines:  2

  Running Processes:
    PID      GOROUTINES   STARTED AT               
    ────────────────────────────────────────────
    7828     2            2026-07-15T20:55:24Z     
```

#### List Enqueued Jobs
```bash
.\queuectl.exe list --state pending
```
**Output:**
```text
ID             COMMAND                   STATE        ATTEMPTS   ERROR
──────────────────────────────────────────────────────────────────────────────────────────
job2           sleep 2                   pending      0/3        -

Total: 1 job(s)
```

#### View Telemetry Stats
```bash
.\queuectl.exe stats
```
**Output:**
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
  Worker Goroutines:     2
==============================================================
```

#### View Output and Error Logs for a Job
```bash
.\queuectl.exe logs job-fail
```
**Output:**
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

### Dead Letter Queue (DLQ) Management

#### List DLQ Jobs
```bash
.\queuectl.exe dlq list
```
**Output:**
```text
Dead Letter Queue
ID             COMMAND                   ATTEMPTS   ERROR                                   
──────────────────────────────────────────────────────────────────────────────────────────
job-fail       non_existent_command      3/3        exit status 1                           

Total: 1 dead job(s)
```

#### Re-queue a Failed Job
```bash
.\queuectl.exe dlq retry job-fail
```
**Output:**
```text
[INFO] Job 'job-fail' moved back to pending. It will be picked up by workers.
```

---

### Monitoring Dashboard
Start the local HTML dashboard server (defaults to port 8080):
```bash
.\queuectl.exe dashboard --port 8080
```
**Output:**
```text
[INFO] QueueCTL Dashboard starting on http://127.0.0.1:8080
```

---

## Architecture Overview

### Job Life Cycle
Jobs traverse through five core states:
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
SQLite coordinates job status and configurations. Write-Ahead Logging (WAL) is enabled by default, and `busy_timeout` is set to 5000ms. This prevents locking during concurrent reads/writes and allows multiple worker processes to access the database safely.

### Worker Execution Flow
1. **Polling**: Workers periodically scan the database for ready jobs (`pending` or retry-ready `failed` jobs whose backoff delay has expired).
2. **Optimistic Claim Locking**: To prevent multiple workers from claiming the same job, workers perform an atomic claim:
   ```sql
   UPDATE jobs SET state = 'processing' WHERE id = ? AND state = ?
   ```
   Only the thread that successfully updates the row (checking if `RowsAffected() > 0`) processes the job.
3. **Execution**: The job's command is executed as a child process. On Windows, it executes via `powershell.exe` to natively support commands like `sleep 2` and `echo`. On Unix, it falls back to `sh`.
4. **Exponential Backoff**: If a command fails, the next execution run time is calculated using:
   $$\text{delay} = \text{backoff\_base}^{\text{attempts}}\text{ seconds}$$
   If `attempts >= max_retries`, the job is marked as `dead` (DLQ).

---

## Assumptions & Trade-offs

### Polling vs. Events
Because SQLite is an embedded, file-based database, it lacks a native pub-sub or event channel mechanism. Workers must poll the database on an interval (defaulting to 2 seconds when idle). The polling interval is small enough to keep latencies low, and the WAL mode ensures database reads do not block enqueuers.

### Cross-Platform Stop Signals
Unix signals (like `SIGUSR1` or custom heartbeat intercepts) behave inconsistently on Windows. We chose a database-backed heartbeat table (`worker_processes`). Stop requests write a `stop_requested` flag to this table, which workers check during heartbeats. This guarantees clean graceful shutdowns across Windows, macOS, and Linux without native OS platform-specific overrides.

---

## Testing Instructions

### Automated Tests
The repository includes unit and integration tests verifying claims, exponential retries, and priority scheduling. Run them with:
```bash
go test -v ./worker
```

### Manual Verification Flow
To test the queue behavior manually:

1. **Verify Success**:
   ```bash
   .\queuectl.exe enqueue --id test-ok --command "echo 'hello'"
   .\queuectl.exe worker start --count 1
   ```
   The worker should execute the echo command and mark it `completed`.

2. **Verify Retries & DLQ**:
   ```bash
   .\queuectl.exe enqueue --id test-fail --command "invalid_command_name"
   ```
   Start the worker. The task will fail, schedule backoff retries, and eventually move to `dead` state (DLQ).

3. **Verify Concurrency Protection**:
   Open two separate command prompts:
   - Term A: `.\queuectl.exe worker start --count 2`
   - Term B: `.\queuectl.exe worker start --count 2`
   
   Enqueue multiple sleep commands and verify they run concurrently across the separate worker processes without double-execution.
