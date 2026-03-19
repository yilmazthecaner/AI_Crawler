package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"spidersearch/internal/crawler"
	"spidersearch/internal/index"
	"strings"
	"time"
)

type WebUI struct {
	Manager   *crawler.JobManager
	FileIndex *index.FileIndex
	Port      int
}

func NewWebUI(jm *crawler.JobManager, fi *index.FileIndex, port int) *WebUI {
	return &WebUI{
		Manager:   jm,
		FileIndex: fi,
		Port:      port,
	}
}

func (w *WebUI) Start() error {
	http.HandleFunc("/", w.handleCrawlerPage)
	http.HandleFunc("/status/", w.handleStatusPage)
	http.HandleFunc("/search", w.handleSearchPage)
	
	http.HandleFunc("/api/jobs", w.handleListJobs)
	http.HandleFunc("/api/create", w.handleCreateJob)
	http.HandleFunc("/api/clear", w.handleClearHistory)
	http.HandleFunc("/api/cancel", w.handleCancelJob)
	http.HandleFunc("/api/status/", w.handleGetStatus)
	http.HandleFunc("/api/search", w.handleSearchAPI)

	fmt.Printf("Web UI available at http://localhost:%d\n", w.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", w.Port), nil)
}

func (w *WebUI) handleCrawlerPage(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprint(rw, crawlerHTML)
}

func (w *WebUI) handleStatusPage(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprint(rw, statusHTML)
}

func (w *WebUI) handleSearchPage(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprint(rw, searchHTML)
}

func (w *WebUI) handleListJobs(rw http.ResponseWriter, r *http.Request) {
	jobs := w.Manager.ListJobs()
	json.NewEncoder(rw).Encode(jobs)
}

func (w *WebUI) handleClearHistory(rw http.ResponseWriter, r *http.Request) {
	w.Manager.ClearJobs()
	fmt.Fprintf(rw, "History cleared")
}

func (w *WebUI) handleCancelJob(rw http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	job := w.Manager.GetJob(id)
	if job != nil {
		job.Cancel()
		fmt.Fprintf(rw, "Job %s cancelled", id)
	} else {
		http.Error(rw, "Job not found", http.StatusNotFound)
	}
}

func (w *WebUI) handleCreateJob(rw http.ResponseWriter, r *http.Request) {
	origin := r.URL.Query().Get("origin")
	depth := 1
	fmt.Sscanf(r.URL.Query().Get("depth"), "%d", &depth)
	
	job := w.Manager.CreateJob(origin, depth, 10)
	job.Start()
	json.NewEncoder(rw).Encode(job)
}

func (w *WebUI) handleGetStatus(rw http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/status/")
	
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			job := w.Manager.GetJob(id)
			json.NewEncoder(rw).Encode(job)
			return
		case <-ticker.C:
			job := w.Manager.GetJob(id)
			if job != nil {
				json.NewEncoder(rw).Encode(job)
				return
			}
		}
	}
}

func (w *WebUI) handleSearchAPI(rw http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results := w.FileIndex.Search(query)
	json.NewEncoder(rw).Encode(results)
}

const commonHead = `
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SpiderSearch</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg: #000000;
            --accent: #0071e3;
            --surface: rgba(28, 28, 30, 0.7);
            --text: #f5f5f7;
            --text-secondary: #86868b;
            --glass: rgba(255, 255, 255, 0.04);
            --border: rgba(255, 255, 255, 0.1);
        }
        * { box-sizing: border-box; }
        body { 
            margin: 0; 
            font-family: 'Inter', -apple-system, system-ui, sans-serif; 
            background: var(--bg); 
            color: var(--text); 
            -webkit-font-smoothing: antialiased;
            overflow-x: hidden;
        }
        .container { max-width: 980px; margin: 0 auto; padding: 40px 20px; }
        nav { 
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            background: rgba(0,0,0,0.8);
            position: sticky;
            top: 0;
            z-index: 100;
            border-bottom: 0.5px solid var(--border);
            padding: 15px 0;
            display: flex;
            justify-content: center;
            gap: 40px;
        }
        nav a { 
            color: var(--text); 
            text-decoration: none; 
            font-size: 14px; 
            font-weight: 400;
            opacity: 0.8;
            transition: opacity 0.2s;
        }
        nav a:hover { opacity: 1; }
        
        h1 { font-size: 48px; font-weight: 600; text-align: center; margin-bottom: 40px; letter-spacing: -0.02em; }
        h2 { font-size: 24px; font-weight: 500; margin-top: 60px; margin-bottom: 20px; }

        .glass-card {
            background: var(--surface);
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            border: 1px solid var(--border);
            border-radius: 20px;
            padding: 30px;
            margin-bottom: 30px;
            transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        }
        .glass-card:hover { transform: translateY(-5px); }

        input {
            width: 100%;
            background: rgba(255,255,255,0.05);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 15px 20px;
            color: white;
            font-size: 16px;
            margin-bottom: 20px;
            outline: none;
            transition: border-color 0.2s;
        }
        input:focus { border-color: var(--accent); }

        button {
            background: var(--accent);
            color: white;
            border: none;
            border-radius: 20px;
            padding: 12px 28px;
            font-size: 16px;
            font-weight: 500;
            cursor: pointer;
            transition: opacity 0.2s, transform 0.1s;
        }
        button:hover { opacity: 0.9; }
        button:active { transform: scale(0.98); }

        .job-item {
            display: grid;
            grid-template-columns: 1fr 320px 100px 80px 100px;
            align-items: center;
            padding: 20px;
            border-bottom: 0.5px solid var(--border);
            gap: 20px;
        }
        .job-item:last-child { border-bottom: none; }
        .status-badge {
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 12px;
            text-transform: uppercase;
            font-weight: 600;
        }
        .status-running { background: rgba(0, 113, 227, 0.1); color: #0071e3; }
        .status-finished { background: rgba(52, 199, 89, 0.1); color: #34c759; }
        .status-error { background: rgba(255, 59, 48, 0.1); color: #ff3b30; }
        .status-cancelled { background: rgba(255, 159, 10, 0.1); color: #ff9f0a; }

        .log-container {
            background: #000;
            border-radius: 12px;
            padding: 20px;
            height: 400px;
            overflow-y: auto;
            font-family: 'SF Mono', 'Menlo', monospace;
            font-size: 13px;
            line-height: 1.6;
            color: #00ff41;
        }
        .stat-box { display: flex; flex-direction: column; min-width: 60px; }
        .stat-val { font-size: 16px; font-weight: 600; color: var(--text); }
        .stat-lbl { font-size: 10px; text-transform: uppercase; color: var(--text-secondary); letter-spacing: 0.05em; margin-top: 2px; }
        
        .result-card {
            padding: 20px;
            border-bottom: 0.5px solid var(--border);
            transition: background 0.2s;
        }
        .result-card:hover { background: rgba(255,255,255,0.02); }
        .result-card a { color: var(--accent); text-decoration: none; font-size: 18px; font-weight: 500; }
        .result-meta { color: var(--text-secondary); font-size: 13px; margin-top: 5px; }
    </style>
</head>
`

const nav = `
<nav>
    <a href="/">Crawler</a>
    <a href="/search">Search Discovery</a>
</nav>
`

const crawlerHTML = `
<!DOCTYPE html>
<html>` + commonHead + `<body>` + nav + `
    <div class="container">
        <h1>SpiderSearch</h1>
        <div class="glass-card">
            <h2 style="margin-top: 0; margin-bottom: 25px;">Create New Job</h2>
            <div style="display: grid; grid-template-columns: 1fr 120px 140px; gap: 20px; align-items: end;">
                <div>
                    <label style="display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 8px;">Origin URL</label>
                    <input type="text" id="origin" placeholder="https://..." style="margin-bottom: 0;">
                </div>
                <div>
                    <label style="display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 8px;">Depth</label>
                    <input type="number" id="depth" value="1" style="margin-bottom: 0;">
                </div>
                <div>
                    <button onclick="createJob()" style="width: 100%; height: 50px; border-radius: 12px; font-weight: 600;">Start Crawl</button>
                </div>
            </div>
        </div>

        <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 40px; margin-bottom: 20px;">
            <h2 style="margin: 0;">Active & History</h2>
            <button onclick="clearHistory()" class="secondary" style="height: 36px; padding: 0 15px; font-size: 13px; border-radius: 8px; background: rgba(255,59,48,0.1); border-color: rgba(255,59,48,0.2); color: #ff453a;">Clear All History</button>
        </div>
        <div id="jobList" class="glass-card" style="padding: 0"></div>
    </div>
    <script>
        async function cancelJob(id) {
            if(!confirm('Stop this job?')) return;
            await fetch('/api/cancel?id=' + id);
            loadJobs();
        }
        async function clearHistory() {
            if(!confirm('Are you sure you want to clear all history? This will also delete all .data files.')) return;
            await fetch('/api/clear');
            loadJobs();
        }
        async function createJob() {
            const o = document.getElementById('origin').value;
            const d = document.getElementById('depth').value;
            if(!o) return;
            await fetch('/api/create?origin=' + encodeURIComponent(o) + '&depth=' + d);
            loadJobs();
        }
        async function loadJobs() {
            const r = await fetch('/api/jobs');
            const jobs = await r.json();
            const container = document.getElementById('jobList');
            container.innerHTML = jobs.length ? '' : '<div style="padding: 40px; text-align: center; color: var(--text-secondary)">No jobs yet.</div>';
            jobs.forEach(j => {
                const div = document.createElement('div');
                div.className = 'job-item';
                
                const startTime = new Date(j.start_time);
                let elapsed = 0;
                if (j.status !== 'running') {
                    elapsed = j.duration;
                } else {
                    elapsed = Math.floor((new Date() - startTime) / 1000);
                }
                const queued = j.total_found - j.crawled_count;
                
                div.innerHTML = ` + "`" + `
                    <div style="min-width: 0">
                        <div style="font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${j.origin_url}</div>
                        <div style="font-size: 11px; color: var(--text-secondary); margin-top: 4px;">ID: ${j.id} • Started: ${startTime.toLocaleTimeString()}</div>
                    </div>
                    
                    <div style="display: flex; gap: 30px; justify-content: center; text-align: center;">
                        <div class="stat-box">
                            <div class="stat-val">${j.crawled_count}</div>
                            <div class="stat-lbl">Crawled</div>
                        </div>
                        <div class="stat-box">
                            <div class="stat-val">${queued > 0 ? queued : 0}</div>
                            <div class="stat-lbl">Queued</div>
                        </div>
                        <div class="stat-box">
                            <div class="stat-val" style="color: ${j.error_count > 0 ? '#ff3b30' : 'inherit'}">${j.error_count}</div>
                            <div class="stat-lbl">Errors</div>
                        </div>
                        <div class="stat-box">
                            <div class="stat-val">${elapsed}s</div>
                            <div class="stat-lbl">Elapsed</div>
                        </div>
                    </div>

                    <div style="text-align: center">
                        <span class="status-badge status-${j.status}" style="display: inline-block; min-width: 80px;">${j.status}</span>
                    </div>

                    <div style="text-align: center">
                        ${j.status === 'running' ? ` + "`" + `<button onclick="cancelJob('${j.id}')" style="background: rgba(255,59,48,0.1); color: #ff3b30; border: none; padding: 6px 12px; border-radius: 6px; font-size: 12px; font-weight: 500; cursor: pointer; width: 100%;">Cancel</button>` + "`" + ` : ''}
                    </div>

                    <div style="text-align: right">
                        <a href="/status/${j.id}" style="color: var(--accent); text-decoration: none; font-size: 14px; white-space: nowrap;">View Logs →</a>
                    </div>
                ` + "`" + `;
                container.appendChild(div);
            });
        }
        setInterval(loadJobs, 2000);
        loadJobs();
    </script>
</body></html>
`

const statusHTML = `
<!DOCTYPE html>
<html>` + commonHead + `<body>` + nav + `
    <div class="container">
        <h1 id="jobTitle">Job Status</h1>
        <div class="glass-card">
            <div id="statusInfo"></div>
            <div id="logs" class="log-container" style="margin-top: 20px"></div>
        </div>
    </div>
    <script>
        const id = window.location.pathname.split('/').pop();
        async function poll() {
            const r = await fetch('/api/status/' + id);
            const job = await r.json();
            if (!job) return;
            
            document.getElementById('jobTitle').innerText = job.origin_url.split('://')[1];
            document.getElementById('statusInfo').innerHTML = ` + "`" + `
                <div style="display: flex; justify-content: space-between; align-items: center">
                    <div>
                        <div style="font-size: 14px; color: var(--text-secondary)">JOB ID</div>
                        <div style="font-size: 18px; font-weight: 600">${job.id}</div>
                    </div>
                    <span class="status-badge status-${job.status}">${job.status}</span>
                </div>
            ` + "`" + `;
            
            const logBox = document.getElementById('logs');
            logBox.innerHTML = job.logs.map(l => ` + "`" + `<div>${l}</div>` + "`" + `).join('');
            logBox.scrollTop = logBox.scrollHeight;
            
            if (job.status === 'running') setTimeout(poll, 1000);
        }
        poll();
    </script>
</body></html>
`

const searchHTML = `
<!DOCTYPE html>
<html>` + commonHead + `<body>` + nav + `
    <div class="container">
        <h1>Global Discovery</h1>
        <div class="glass-card">
            <div style="display: flex; gap: 15px">
                <input type="text" id="q" placeholder="Search keywords..." onkeypress="if(event.key==='Enter') doSearch()" style="margin-bottom: 0">
                <button onclick="doSearch()">Search</button>
            </div>
        </div>
        <div id="results" class="glass-card" style="padding: 0; display: none;"></div>
    </div>
    <script>
        async function doSearch() {
            const q = document.getElementById('q').value;
            if(!q) return;
            const container = document.getElementById('results');
            container.style.display = 'block';
            container.innerHTML = '<div style="padding: 40px; text-align: center; color: var(--text-secondary)">Searching the filesystem database...</div>';
            
            try {
                const r = await fetch('/api/search?q=' + encodeURIComponent(q));
                const data = await r.json();
                container.innerHTML = '';
                
                if (!data || data.length === 0) {
                    container.innerHTML = ` + "`" + `
                        <div style="padding: 60px 40px; text-align: center;">
                            <div style="font-size: 48px; margin-bottom: 20px;">🔍</div>
                            <div style="font-size: 20px; font-weight: 500; color: var(--text);">No results found for "${q}"</div>
                            <div style="color: var(--text-secondary); margin-top: 10px;">Try different keywords or check if the crawler job is finished.</div>
                        </div>
                    ` + "`" + `;
                    return;
                }
                
                data.sort((a,b) => b.relevance - a.relevance);
                data.forEach(res => {
                    const div = document.createElement('div');
                    div.className = 'result-card';
                    div.innerHTML = ` + "`" + `
                        <a href="${res.url}" target="_blank">${res.url}</a>
                        <div class="result-meta">
                            Score: <strong>${Math.round(res.relevance)}</strong> • 
                            Origin: ${res.origin_url} • 
                            Depth: ${res.depth}
                        </div>
                    ` + "`" + `;
                    container.appendChild(div);
                });
            } catch (e) {
                container.innerHTML = '<div style="padding: 40px; text-align: center; color: #ff3b30">Error connecting to the search API.</div>';
            }
        }
    </script>
</body></html>
`
