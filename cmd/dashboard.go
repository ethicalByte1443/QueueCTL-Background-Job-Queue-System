package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ethicalByte1443/queuectl/db"
	"github.com/spf13/cobra"
)

var dashboardPort int

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start a web dashboard for queue monitoring",
	Long:  `Launch a local web server serving a clean, raw HTML/CSS technical table dashboard to monitor jobs, workers, and metrics in real-time.`,
	Run: func(cmd *cobra.Command, args []string) {
		http.HandleFunc("/", handleHome)
		http.HandleFunc("/api/data", handleData)
		http.HandleFunc("/api/retry", handleRetry)

		addr := fmt.Sprintf("127.0.0.1:%d", dashboardPort)
		fmt.Printf("[INFO] QueueCTL Dashboard starting on http://%s\n", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Server failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	dashboardCmd.Flags().IntVarP(&dashboardPort, "port", "p", 8080, "Port to run the dashboard web server on")
}

type DashboardData struct {
	Stats struct {
		Total       int     `json:"total"`
		Pending     int     `json:"pending"`
		Processing  int     `json:"processing"`
		Completed   int     `json:"completed"`
		Failed      int     `json:"failed"`
		Dead        int     `json:"dead"`
		SuccessRate float64 `json:"success_rate"`
		AvgDuration float64 `json:"avg_duration"`
	} `json:"stats"`
	Workers []struct {
		PID         int    `json:"pid"`
		WorkerCount int    `json:"worker_count"`
		StartedAt   string `json:"started_at"`
		UpdatedAt   string `json:"updated_at"`
	} `json:"workers"`
	Jobs []struct {
		ID         string `json:"id"`
		Command    string `json:"command"`
		State      string `json:"state"`
		Attempts   int    `json:"attempts"`
		MaxRetries int    `json:"max_retries"`
		Priority   int    `json:"priority"`
		RunAt      string `json:"run_at"`
		Timeout    int    `json:"timeout"`
		DurationMs int    `json:"duration_ms"`
		ErrorMsg   string `json:"error_msg"`
		Output     string `json:"output"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	} `json:"jobs"`
}

func handleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var data DashboardData
	data.Workers = make([]struct {
		PID         int    `json:"pid"`
		WorkerCount int    `json:"worker_count"`
		StartedAt   string `json:"started_at"`
		UpdatedAt   string `json:"updated_at"`
	}, 0)
	data.Jobs = make([]struct {
		ID         string `json:"id"`
		Command    string `json:"command"`
		State      string `json:"state"`
		Attempts   int    `json:"attempts"`
		MaxRetries int    `json:"max_retries"`
		Priority   int    `json:"priority"`
		RunAt      string `json:"run_at"`
		Timeout    int    `json:"timeout"`
		DurationMs int    `json:"duration_ms"`
		ErrorMsg   string `json:"error_msg"`
		Output     string `json:"output"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	}, 0)

	// Fetch Stats counts
	rows, err := db.DB.Query("SELECT state, COUNT(*) FROM jobs GROUP BY state")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err == nil {
			data.Stats.Total += count
			switch state {
			case "pending":
				data.Stats.Pending = count
			case "processing":
				data.Stats.Processing = count
			case "completed":
				data.Stats.Completed = count
			case "failed":
				data.Stats.Failed = count
			case "dead":
				data.Stats.Dead = count
			}
		}
	}

	if data.Stats.Total > 0 {
		data.Stats.SuccessRate = (float64(data.Stats.Completed) / float64(data.Stats.Total)) * 100
	}

	// Fetch Avg Duration
	_ = db.DB.QueryRow("SELECT COALESCE(AVG(duration_ms), 0) FROM jobs WHERE state = 'completed'").Scan(&data.Stats.AvgDuration)

	// Fetch Active Worker Processes
	workerRows, err := db.DB.Query(`
		SELECT pid, worker_count, started_at, updated_at
		FROM worker_processes
		WHERE status = 'running'
		  AND updated_at >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-10 seconds')
	`)
	if err == nil {
		defer workerRows.Close()
		for workerRows.Next() {
			var wp struct {
				PID         int    `json:"pid"`
				WorkerCount int    `json:"worker_count"`
				StartedAt   string `json:"started_at"`
				UpdatedAt   string `json:"updated_at"`
			}
			if err := workerRows.Scan(&wp.PID, &wp.WorkerCount, &wp.StartedAt, &wp.UpdatedAt); err == nil {
				data.Workers = append(data.Workers, wp)
			}
		}
	}

	// Fetch Jobs
	jobRows, err := db.DB.Query(`
		SELECT id, command, state, attempts, max_retries, priority, COALESCE(run_at, ''), timeout, duration_ms, COALESCE(error_msg, ''), COALESCE(output, ''), created_at, updated_at
		FROM jobs
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err == nil {
		defer jobRows.Close()
		for jobRows.Next() {
			var j struct {
				ID         string `json:"id"`
				Command    string `json:"command"`
				State      string `json:"state"`
				Attempts   int    `json:"attempts"`
				MaxRetries int    `json:"max_retries"`
				Priority   int    `json:"priority"`
				RunAt      string `json:"run_at"`
				Timeout    int    `json:"timeout"`
				DurationMs int    `json:"duration_ms"`
				ErrorMsg   string `json:"error_msg"`
				Output     string `json:"output"`
				CreatedAt  string `json:"created_at"`
				UpdatedAt  string `json:"updated_at"`
			}
			if err := jobRows.Scan(
				&j.ID, &j.Command, &j.State, &j.Attempts, &j.MaxRetries, &j.Priority, &j.RunAt, &j.Timeout, &j.DurationMs, &j.ErrorMsg, &j.Output, &j.CreatedAt, &j.UpdatedAt,
			); err == nil {
				data.Jobs = append(data.Jobs, j)
			}
		}
	}

	json.NewEncoder(w).Encode(data)
}

func handleRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}
	query := `UPDATE jobs SET state = 'pending', attempts = 0, error_msg = '', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ? AND state = 'dead'`
	result, err := db.DB.Exec(query, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Job not found in DLQ", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlPage))
}

const htmlPage = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>QueueCTL Admin Console</title>
    <style>
        body {
            font-family: monospace;
            padding: 20px;
            background-color: #ffffff;
            color: #000000;
        }
        table {
            border-collapse: collapse;
            width: 100%;
            margin-bottom: 20px;
        }
        th, td {
            border: 1px solid #000000;
            padding: 6px 10px;
            text-align: left;
            font-size: 12px;
        }
        th {
            background-color: #f0f0f0;
        }
        tr.clickable:hover td {
            background-color: #f9f9f9;
            cursor: pointer;
        }
        .nav {
            margin-bottom: 20px;
            font-size: 13px;
        }
        .nav a {
            text-decoration: none;
            color: blue;
            margin-right: 15px;
        }
        .nav a.active {
            font-weight: bold;
            color: black;
            text-decoration: underline;
        }
        .btn-link {
            background: none;
            border: none;
            color: blue;
            text-decoration: underline;
            cursor: pointer;
            padding: 0;
            font-family: monospace;
            font-size: 12px;
        }
        .detail-pane {
            border: 2px solid #000000;
            padding: 15px;
            margin-top: 20px;
            background-color: #fcfcfc;
        }
    </style>
</head>
<body>
    <h1>QueueCTL Admin Console</h1>
    <hr>
    
    <div class="nav">
        <strong>Filter State:</strong>
        <a href="#" id="tab-all" class="active" onclick="setFilter('all')">[All]</a>
        <a href="#" id="tab-pending" onclick="setFilter('pending')">[Pending]</a>
        <a href="#" id="tab-processing" onclick="setFilter('processing')">[Processing]</a>
        <a href="#" id="tab-completed" onclick="setFilter('completed')">[Completed]</a>
        <a href="#" id="tab-failed" onclick="setFilter('failed')">[Failed]</a>
        <a href="#" id="tab-dead" onclick="setFilter('dead')">[DLQ (Dead)]</a>
    </div>

    <h2>Jobs Queue (<span id="jobs-count">0</span>)</h2>
    <div style="overflow-x: auto;">
        <table>
            <thead>
                <tr>
                    <th>Job ID</th>
                    <th>Command</th>
                    <th>State</th>
                    <th>Priority</th>
                    <th>Attempts</th>
                    <th>Duration</th>
                    <th>Updated At</th>
                    <th>Action</th>
                </tr>
            </thead>
            <tbody id="jobs-table-body">
                <tr><td colspan="8">Loading data...</td></tr>
            </tbody>
        </table>
    </div>

    <div style="display: flex; gap: 40px; margin-top: 30px; flex-wrap: wrap;">
        <div style="flex: 1; min-width: 300px;">
            <h2>Active Worker Processes</h2>
            <table>
                <thead>
                    <tr>
                        <th>PID</th>
                        <th>Goroutines</th>
                        <th>Started At</th>
                        <th>Last Heartbeat</th>
                    </tr>
                </thead>
                <tbody id="workers-table-body">
                    <tr><td colspan="4">No active worker processes</td></tr>
                </tbody>
            </table>
        </div>
        <div style="width: 350px;">
            <h2>System Statistics</h2>
            <table>
                <tbody>
                    <tr><td>Total Enqueued:</td><td id="stat-total" style="font-weight: bold;">0</td></tr>
                    <tr><td>Pending:</td><td id="stat-pending">0</td></tr>
                    <tr><td>Processing:</td><td id="stat-processing">0</td></tr>
                    <tr><td>Completed:</td><td id="stat-completed">0</td></tr>
                    <tr><td>Failed (Retryable):</td><td id="stat-failed">0</td></tr>
                    <tr><td>Dead (DLQ):</td><td id="stat-dead">0</td></tr>
                    <tr><td>Success Rate:</td><td id="metric-success-rate">0.00%</td></tr>
                    <tr><td>Avg Run Duration:</td><td id="metric-avg-duration">0 ms</td></tr>
                </tbody>
            </table>
        </div>
    </div>

    <div id="details-section" class="detail-pane" style="display: none;">
        <h2 style="margin-top: 0;">Job Details: <span id="detail-job-id"></span></h2>
        <table>
            <tbody>
                <tr><td style="width: 150px; font-weight: bold;">Command:</td><td id="detail-command"></td></tr>
                <tr><td style="font-weight: bold;">State:</td><td id="detail-state"></td></tr>
                <tr><td style="font-weight: bold;">Priority:</td><td id="detail-priority"></td></tr>
                <tr><td style="font-weight: bold;">Attempts:</td><td id="detail-attempts"></td></tr>
                <tr><td style="font-weight: bold;">Duration:</td><td id="detail-duration"></td></tr>
                <tr><td style="font-weight: bold;">Created At:</td><td id="detail-created-at"></td></tr>
                <tr><td style="font-weight: bold;">Updated At:</td><td id="detail-updated-at"></td></tr>
            </tbody>
        </table>
        
        <div id="detail-error-box" style="margin-top: 15px; display: none;">
            <strong>Error Verdict Logs:</strong>
            <pre id="detail-error" style="border: 1px solid #ff0000; background-color: #fff8f8; padding: 10px; margin-top: 5px; white-space: pre-wrap; word-break: break-all;"></pre>
        </div>

        <div style="margin-top: 15px;">
            <strong>Console Output Logs:</strong>
            <pre id="detail-output" style="border: 1px solid #cccccc; background-color: #f9f9f9; padding: 10px; margin-top: 5px; white-space: pre-wrap; word-break: break-all;"></pre>
        </div>
        <br>
        <button class="btn-link" style="color: red; font-weight: bold;" onclick="document.getElementById('details-section').style.display='none'">[Close Details Panel]</button>
    </div>

    <script>
        let currentFilter = 'all';
        let jobsData = [];

        async function refreshData() {
            try {
                const res = await fetch('/api/data');
                const data = await res.json();
                
                // Update stats table
                document.getElementById('stat-total').innerText = data.stats.total;
                document.getElementById('stat-pending').innerText = data.stats.pending;
                document.getElementById('stat-processing').innerText = data.stats.processing;
                document.getElementById('stat-completed').innerText = data.stats.completed;
                document.getElementById('stat-failed').innerText = data.stats.failed;
                document.getElementById('stat-dead').innerText = data.stats.dead;

                // Update analytics metrics
                document.getElementById('metric-success-rate').innerText = data.stats.success_rate.toFixed(2) + '%';
                document.getElementById('metric-avg-duration').innerText = Math.round(data.stats.avg_duration) + ' ms';

                // Update Workers list table
                const workersTableBody = document.getElementById('workers-table-body');
                workersTableBody.innerHTML = '';
                
                if (data.workers.length === 0) {
                    workersTableBody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: gray;">No active worker processes</td></tr>';
                } else {
                    data.workers.forEach(w => {
                        workersTableBody.innerHTML += 
                            '<tr>' +
                                '<td>' + w.pid + '</td>' +
                                '<td>' + w.worker_count + '</td>' +
                                '<td>' + formatDate(w.started_at) + '</td>' +
                                '<td>' + formatDate(w.updated_at) + '</td>' +
                            '</tr>';
                    });
                }

                // Update Jobs list table
                jobsData = data.jobs;
                renderJobsTable();
            } catch (err) {
                console.error("Failed to load dashboard data:", err);
            }
        }

        function setFilter(filter) {
            currentFilter = filter;
            
            const tabs = ['all', 'pending', 'processing', 'completed', 'failed', 'dead'];
            tabs.forEach(t => {
                const link = document.getElementById('tab-' + t);
                if (link) {
                    if (t === filter) {
                        link.className = 'active';
                    } else {
                        link.className = '';
                    }
                }
            });

            renderJobsTable();
        }

        function renderJobsTable() {
            const tableBody = document.getElementById('jobs-table-body');
            const countBadge = document.getElementById('jobs-count');
            
            const filteredJobs = jobsData.filter(j => {
                if (currentFilter === 'all') return true;
                return j.state === currentFilter;
            });

            countBadge.innerText = filteredJobs.length;
            tableBody.innerHTML = '';

            filteredJobs.forEach(j => {
                const actionBtn = j.state === 'dead' 
                    ? '<button class="btn-link" onclick="retryJob(event, \'' + j.id + '\')">[Retry]</button>'
                    : '-';

                tableBody.innerHTML += 
                    '<tr class="clickable" onclick="showJobDetails(\'' + j.id + '\')">' +
                        '<td style="font-weight: bold;">' + j.id + '</td>' +
                        '<td>' + escapeHTML(j.command) + '</td>' +
                        '<td>' + j.state + '</td>' +
                        '<td>' + j.priority + '</td>' +
                        '<td>' + j.attempts + '/' + j.max_retries + '</td>' +
                        '<td>' + j.duration_ms + ' ms</td>' +
                        '<td>' + formatDate(j.updated_at) + '</td>' +
                        '<td>' + actionBtn + '</td>' +
                    '</tr>';
            });

            if (filteredJobs.length === 0) {
                tableBody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: gray;">No jobs in this filter state.</td></tr>';
            }
        }

        async function retryJob(event, id) {
            event.stopPropagation();
            if (!confirm("Retry dead job '" + id + "'?")) return;

            try {
                const res = await fetch('/api/retry?id=' + encodeURIComponent(id), { method: 'POST' });
                if (res.ok) {
                    refreshData();
                } else {
                    const text = await res.text();
                    alert("Failed to retry job: " + text);
                }
            } catch (err) {
                alert("Failed to retry job: " + err);
            }
        }

        function showJobDetails(id) {
            const job = jobsData.find(j => j.id === id);
            if (!job) return;

            document.getElementById('detail-job-id').innerText = job.id;
            document.getElementById('detail-command').innerText = job.command;
            document.getElementById('detail-state').innerText = job.state;
            document.getElementById('detail-priority').innerText = job.priority;
            document.getElementById('detail-attempts').innerText = job.attempts + '/' + job.max_retries;
            document.getElementById('detail-duration').innerText = job.duration_ms + ' ms';
            
            const runAtField = document.getElementById('modal-run-at-field');
            if (job.run_at) {
                document.getElementById('modal-run-at').innerText = formatDate(job.run_at);
                runAtField.style.display = 'table-row';
            } else {
                runAtField.style.display = 'none';
            }

            document.getElementById('detail-created-at').innerText = formatDate(job.created_at);
            document.getElementById('detail-updated-at').innerText = formatDate(job.updated_at);

            const errSection = document.getElementById('detail-error-box');
            if (job.error_msg) {
                document.getElementById('detail-error').innerText = job.error_msg;
                errSection.style.display = 'block';
            } else {
                errSection.style.display = 'none';
            }

            document.getElementById('detail-output').innerText = job.output || "(No stdout log output recorded)";

            document.getElementById('details-section').style.display = 'block';
            document.getElementById('details-section').scrollIntoView({ behavior: 'smooth' });
        }

        function formatDate(isoStr) {
            if (!isoStr) return '-';
            try {
                const date = new Date(isoStr);
                return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) + ' ' + date.toLocaleDateString();
            } catch (e) {
                return isoStr;
            }
        }

        function escapeHTML(str) {
            return str.replace(/[&<>'"]/g, 
                tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
            );
        }

        // Initialize and setup polling
        refreshData();
        setInterval(refreshData, 2000);
    </script>
</body>
</html>
`
