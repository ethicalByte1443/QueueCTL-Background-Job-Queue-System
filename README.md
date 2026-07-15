# QueueCTL — Background Job Queue System

## Project Status: 🚧 In Progress

QueueCTL is a CLI-based background job queue system that manages background tasks with worker processes, automatic retries using exponential backoff, and a Dead Letter Queue (DLQ) for permanently failed jobs.

---

## Tech Stack Details

* **Language:** Go (Golang)
* **CLI Framework:** [Cobra](https://github.com/spf13/cobra) (Industry-standard library for modern sub-command CLIs)
* **Storage / Persistence:** SQLite (Ensures ACID compliance, robust file-locking, and zero external database setup)

---

## System Flow & Architecture

```
                       [ queuectl enqueue ]
                                │
                                ▼
                       ( SQLite Database )
                                │
          ┌─────────────────────┴─────────────────────┐
          ▼ (Worker 1)                                ▼ (Worker 2)
  [ Lock & Pick Job ]                         [ Lock & Pick Job ]
          │                                           │
          ▼                                           ▼
   [ Execute Command ]                         [ Execute Command ]
          │                                           │
          ├──► Success ──► [ completed ]              ├──► Success ──► [ completed ]
          │                                           │
          └──► Failure ──► Retries exhausted?         └──► Failure ──► Retries exhausted?
                               │                                           │
                               ├──► Yes ──► [ dead ]                       ├──► Yes ──► [ dead ]
                               │                                           │
                               └──► No  ──► [ failed ] (Backoff wait)      └──► No  ──► [ failed ]
```

---

## Algorithms

### 1. Storing Jobs (Enqueue Stage)
When a user enqueues a job via `queuectl enqueue`, the application:
1. Validates the JSON payload containing the job `id` and shell `command`.
2. Inserts the job into the SQLite database with:
   * `state` = `pending`
   * `attempts` = `0`
   * `max_retries` = Configured limit (default: 3)
   * `created_at` and `updated_at` = current timestamp.

### 2. Lock & Pick Jobs (Worker Stage)
To support multiple workers in parallel without duplicate processing (race conditions), we use a database transaction with write-locking:
1. **Transaction Begin:** Open a `BEGIN IMMEDIATE` transaction in SQLite to write-lock the database.
2. **Select Candidate:** Search for a single job that matches either of these conditions:
   * `state = 'pending'`
   * `state = 'failed'` **AND** current time ≥ `updated_at` + (base^attempts) seconds (exponential backoff check).
3. **Atomic Claim:** Update the selected job's `state` to `processing` and update `updated_at` to the current time.
4. **Transaction Commit:** Release the lock.
5. **Execution:** The worker executes the command using Go's `os/exec`.

### 3. Finish / Retry Stage
Once execution completes:
* **If Exit Code = 0 (Success):** The worker updates the job's `state` to `completed`.
* **If Exit Code != 0 (Failure):**
  * The worker increments the `attempts` count.
  * If `attempts < max_retries`: Set state to `failed` and update `updated_at` (triggers exponential backoff delay before the next attempt).
  * If `attempts >= max_retries`: Move state to `dead` (DLQ).

---

## Git Branching Strategy

We follow a **Git Flow** process:
* `main` — Production-ready code
* `develop` — Integration branch where all features merge
* `feature/*` — Individual task branches created from `develop`

---

## License

MIT License