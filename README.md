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

## License
MIT License