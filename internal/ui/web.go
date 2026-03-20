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
	
	hitRate := 10
	fmt.Sscanf(r.URL.Query().Get("hitRate"), "%d", &hitRate)

	queueCap := 0
	fmt.Sscanf(r.URL.Query().Get("queueCap"), "%d", &queueCap)

	maxURLs := 0
	fmt.Sscanf(r.URL.Query().Get("maxURLs"), "%d", &maxURLs)
	
	job := w.Manager.CreateJob(origin, depth, 10, hitRate, queueCap, maxURLs)
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
    <title>SpiderSearch | Premium Crawler</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg: #0b0b0d;
            --surface: rgba(28, 28, 30, 0.6);
            --surface-hover: rgba(44, 44, 46, 0.8);
            --accent: #0a84ff;
            --accent-gradient: linear-gradient(135deg, #0a84ff, #5e5ce6);
            --text: #ffffff;
            --text-secondary: #a1a1aa;
            --border: rgba(255, 255, 255, 0.08);
            --glass: blur(16px);
            --error: #ff453a;
            --success: #32d74b;
            --warning: #ff9f0a;
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            background-color: var(--bg);
            background-image: 
                radial-gradient(circle at 0% 0%, rgba(10, 132, 255, 0.05) 0%, transparent 50%),
                radial-gradient(circle at 100% 100%, rgba(94, 92, 230, 0.05) 0%, transparent 50%);
            background-attachment: fixed;
            color: var(--text);
            font-family: 'Inter', -apple-system, sans-serif;
            -webkit-font-smoothing: antialiased;
            line-height: 1.5;
        }

        .container { max-width: 1100px; margin: 0 auto; padding: 40px 24px; }

        nav {
            position: sticky;
            top: 20px;
            z-index: 1000;
            display: flex;
            justify-content: center;
            gap: 8px;
            margin-bottom: 40px;
        }

        nav a {
            background: var(--surface);
            backdrop-filter: var(--glass);
            -webkit-backdrop-filter: var(--glass);
            border: 1px solid var(--border);
            padding: 10px 20px;
            border-radius: 100px;
            color: var(--text-secondary);
            text-decoration: none;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        }

        nav a:hover {
            color: var(--text);
            background: var(--surface-hover);
            transform: translateY(-1px);
        }

        nav a.active {
            background: var(--accent);
            color: white;
            border-color: transparent;
            box-shadow: 0 4px 12px rgba(10, 132, 255, 0.3);
        }

        h1 {
            font-size: 42px;
            font-weight: 700;
            text-align: center;
            margin-bottom: 30px;
            letter-spacing: -0.04em;
            background: linear-gradient(to bottom, #fff, #a1a1aa);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;
            margin-bottom: 32px;
        }

        .stat-card {
            background: var(--surface);
            backdrop-filter: var(--glass);
            -webkit-backdrop-filter: var(--glass);
            border: 1px solid var(--border);
            padding: 20px;
            border-radius: 20px;
            text-align: center;
        }

        .stat-value { font-size: 28px; font-weight: 700; color: var(--accent); margin-bottom: 4px; }
        .stat-label { font-size: 12px; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.05em; }

        .card {
            background: var(--surface);
            backdrop-filter: var(--glass);
            -webkit-backdrop-filter: var(--glass);
            border: 1px solid var(--border);
            border-radius: 24px;
            padding: 32px;
            margin-bottom: 32px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
        }

        .form-grid {
            display: grid;
            grid-template-columns: 2fr 1fr;
            gap: 20px;
            margin-bottom: 20px;
        }

        .form-group { margin-bottom: 16px; }
        .form-group label {
            display: block;
            font-size: 13px;
            font-weight: 600;
            color: var(--text-secondary);
            margin-bottom: 8px;
            margin-left: 4px;
        }

        input {
            width: 100%;
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid var(--border);
            border-radius: 14px;
            padding: 14px 18px;
            color: white;
            font-size: 15px;
            font-family: inherit;
            transition: all 0.2s;
        }

        input:focus {
            outline: none;
            border-color: var(--accent);
            background: rgba(255, 255, 255, 0.08);
            box-shadow: 0 0 0 4px rgba(10, 132, 255, 0.1);
        }

        button {
            background: var(--accent-gradient);
            color: white;
            border: none;
            border-radius: 14px;
            padding: 14px 28px;
            font-size: 15px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }

        button:hover {
            transform: translateY(-1px);
            opacity: 0.9;
            box-shadow: 0 4px 15px rgba(10, 132, 255, 0.4);
        }

        button:active { transform: translateY(0); }

        button.secondary {
            background: rgba(255, 255, 255, 0.05);
            color: var(--text);
            border: 1px solid var(--border);
        }

        button.danger {
            background: rgba(255, 69, 58, 0.1);
            color: var(--error);
            border: 1px solid rgba(255, 69, 58, 0.2);
        }
        button.danger:hover {
            background: var(--error);
            color: white;
            box-shadow: 0 4px 15px rgba(255, 69, 58, 0.4);
        }

        .job-card {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: 20px;
            padding: 24px;
            margin-bottom: 16px;
            display: grid;
            grid-template-columns: 1fr auto auto auto;
            align-items: center;
            gap: 32px;
            transition: all 0.3s;
        }
        .job-card:hover {
            background: var(--surface-hover);
            border-color: rgba(255,255,255,0.15);
        }

        .url-text {
            font-weight: 600;
            font-size: 16px;
            margin-bottom: 4px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }
        .meta-text { font-size: 12px; color: var(--text-secondary); }

        .badge {
            padding: 6px 14px;
            border-radius: 100px;
            font-size: 11px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.02em;
        }
        .badge-running { background: rgba(10, 132, 255, 0.15); color: var(--accent); }
        .badge-finished { background: rgba(50, 215, 75, 0.15); color: var(--success); }
        .badge-error { background: rgba(255, 69, 58, 0.15); color: var(--error); }
        .badge-cancelled { background: rgba(161, 161, 170, 0.15); color: var(--text-secondary); }

        .log-console {
            background: #000;
            border-radius: 16px;
            padding: 20px;
            height: 450px;
            overflow-y: auto;
            font-family: 'SF Mono', 'Menlo', monospace;
            font-size: 13px;
            line-height: 1.6;
            color: #d1d1d6;
            border: 1px solid var(--border);
        }
        .log-entry { margin-bottom: 4px; }
        .log-timestamp { color: var(--text-secondary); margin-right: 8px; }

        .search-result {
            padding: 24px;
            border-bottom: 1px solid var(--border);
            transition: all 0.2s;
        }
        .search-result:last-child { border-bottom: none; }
        .search-result:hover { background: rgba(255,255,255,0.02); }
        .search-result-url { color: var(--accent); font-size: 18px; font-weight: 600; text-decoration: none; display: block; margin-bottom: 8px; }
        .search-result-url:hover { text-decoration: underline; }
        .search-result-meta { font-size: 13px; color: var(--text-secondary); display: flex; gap: 16px; }
        .search-result-meta b { color: var(--text); }
    </style>
</head>
`

const nav = `
<nav>
    <a href="/" id="nav-crawler">Spider Intelligence</a>
    <a href="/search" id="nav-discovery">Global Discovery</a>
</nav>
`

const crawlerHTML = `
<!DOCTYPE html>
<html>` + commonHead + `<body>` + nav + `
    <div class="container">
        <h1 style="margin-bottom: 10px;">🕷️ SpiderSearch</h1>
        <p style="text-align: center; color: var(--text-secondary); margin-bottom: 40px; font-size: 15px;">Advanced Distributed Crawler Engine</p>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value" id="stat-visited">0</div>
                <div class="stat-label">URLs Visited</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-active">0</div>
                <div class="stat-label">Active crawlers</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-total">0</div>
                <div class="stat-label">Total Jobs</div>
            </div>
        </div>

        <div class="card">
            <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px;">
                <h2 style="font-size: 20px; font-weight: 600;">🚀 Create New Crawler</h2>
                <div style="display: flex; gap: 8px;">
                     <button onclick="clearHistory()" class="danger" style="padding: 8px 16px; font-size: 12px; border-radius: 10px;">Clear All Data</button>
                </div>
            </div>
            
            <div class="form-grid">
                <div class="form-group">
                    <label>Origin URL</label>
                    <input type="text" id="origin" placeholder="https://www.example.com">
                </div>
                <div class="form-group">
                    <label>Depth (k)</label>
                    <input type="number" id="depth" value="2" min="1" max="10">
                </div>
            </div>

            <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 20px; margin-bottom: 24px;">
                <div class="form-group">
                    <label>Hit Rate (/sec)</label>
                    <input type="number" id="hitRate" value="5" min="1">
                    <p style="font-size: 11px; color: var(--text-secondary); margin-top: 6px;">Request rate limiting (Back pressure)</p>
                </div>
                <div class="form-group">
                    <label>Queue Capacity</label>
                    <input type="number" id="queueCap" value="1000" min="0">
                    <p style="font-size: 11px; color: var(--text-secondary); margin-top: 6px;">0 for unlimited</p>
                </div>
                <div class="form-group">
                    <label>Max URLs to Visit</label>
                    <input type="number" id="maxURLs" value="100" min="1">
                    <p style="font-size: 11px; color: var(--text-secondary); margin-top: 6px;">Hard stop after N pages</p>
                </div>
            </div>

            <button onclick="createJob()" style="width: 100%; height: 54px; font-size: 16px;">
                Initialize Neural Crawl Sequence
            </button>
        </div>

        <h2 style="font-size: 20px; font-weight: 600; margin-bottom: 20px; margin-left: 4px;">Recent Sequences</h2>
        <div id="jobList"></div>
    </div>

    <script>
        document.getElementById('nav-crawler').classList.add('active');

        async function createJob() {
            const btn = event.target;
            const originalText = btn.innerText;
            btn.innerText = 'Initializing...';
            btn.disabled = true;

            const params = new URLSearchParams({
                origin: document.getElementById('origin').value,
                depth: document.getElementById('depth').value,
                hitRate: document.getElementById('hitRate').value,
                queueCap: document.getElementById('queueCap').value,
                maxURLs: document.getElementById('maxURLs').value
            });

            try {
                const r = await fetch('/api/create?' + params.toString());
                if (r.ok) {
                    document.getElementById('origin').value = '';
                    await loadJobs();
                }
            } finally {
                btn.innerText = originalText;
                btn.disabled = false;
            }
        }

        async function cancelJob(id) {
            if(!confirm('Terminate this crawl sequence?')) return;
            await fetch('/api/cancel?id=' + id);
            loadJobs();
        }

        async function clearHistory() {
            if(!confirm('ERASE ALL DATA? This cannot be undone.')) return;
            await fetch('/api/clear');
            loadJobs();
        }

        async function loadJobs() {
            const r = await fetch('/api/jobs');
            const jobs = await r.json();
            
            const list = document.getElementById('jobList');
            const stats = { visited: 0, active: 0, total: jobs.length };

            if (!jobs.length) {
                list.innerHTML = '<div style="text-align: center; padding: 60px; color: var(--text-secondary); background: var(--surface); border-radius: 24px; border: 1px dashed var(--border);">No crawl sequences detected in history.</div>';
            } else {
                list.innerHTML = jobs.map(j => {
                    stats.visited += j.crawled_count;
                    if (j.status === 'running') stats.active++;
                    
                    const percent = j.max_urls > 0 ? Math.min(100, Math.round((j.crawled_count / j.max_urls) * 100)) : 0;
                    
                    return ` + "`" + `
                        <div class="job-card" onclick="window.location.href='/status/${j.id}'" style="cursor: pointer">
                            <div>
                                <div class="url-text">${j.origin_url}</div>
                                <div class="meta-text">ID: ${j.id} • ${new Date(j.start_time).toLocaleString()}</div>
                            </div>
                            
                            <div style="text-align: center">
                                <div style="font-size: 20px; font-weight: 700;">${j.crawled_count}</div>
                                <div class="meta-text">VISITED</div>
                            </div>

                            <div style="width: 100px; text-align: center;">
                                <span class="badge badge-${j.status}">${j.status}</span>
                            </div>

                            <div style="display: flex; gap: 8px;">
                                ${j.status === 'running' ? ` + "`" + `<button onclick="event.stopPropagation(); cancelJob('${j.id}')" class="danger" style="padding: 8px 16px; font-size: 11px;">Stop</button>` + "`" + ` : '<span style="color: var(--text-secondary); font-size: 13px;">View Results →</span>'}
                            </div>
                        </div>
                    ` + "`" + `;
                }).join('');
            }

            document.getElementById('stat-visited').innerText = stats.visited.toLocaleString();
            document.getElementById('stat-active').innerText = stats.active;
            document.getElementById('stat-total').innerText = stats.total;
        }

        setInterval(loadJobs, 2000);
        loadJobs();
    </script>
</body></html>
`

const statusHTML = `
<!DOCTYPE html>
<html>` + commonHead + `<body>` + nav + `
    <div class="container" id="status-container">
        <div style="margin-bottom: 30px; display: flex; align-items: center; gap: 16px;">
            <a href="/" style="text-decoration: none; color: var(--text-secondary); font-size: 24px;">←</a>
            <h1 style="margin-bottom: 0; text-align: left;" id="job-title">Crawl Sequence</h1>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value" id="s-visited">0</div>
                <div class="stat-label">URLs Visited</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="s-queued">0</div>
                <div class="stat-label">In Queue</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="s-errors">0</div>
                <div class="stat-label">Errors</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="s-status">-</div>
                <div class="stat-label">Status</div>
            </div>
        </div>

        <div class="card">
            <h2 style="font-size: 18px; margin-bottom: 20px;">Telemetry Logs</h2>
            <div id="logs" class="log-console"></div>
        </div>
    </div>

    <script>
        const id = window.location.pathname.split('/').pop();
        
        async function poll() {
            const r = await fetch('/api/status/' + id);
            const j = await r.json();
            if (!j) return;
            
            document.getElementById('job-title').innerText = j.origin_url.split('://')[1];
            document.getElementById('s-visited').innerText = j.crawled_count;
            document.getElementById('s-queued').innerText = (j.total_found - j.crawled_count);
            document.getElementById('s-errors').innerText = j.error_count;
            document.getElementById('s-status').innerText = j.status.toUpperCase();
            
            const logBox = document.getElementById('logs');
            const atBottom = logBox.scrollHeight - logBox.scrollTop <= logBox.clientHeight + 50;
            
            logBox.innerHTML = j.logs.map(l => {
                const parts = l.split('] ');
                const ts = parts[0].replace('[', '');
                const msg = parts.slice(1).join('] ');
                return ` + "`" + `<div class="log-entry"><span class="log-timestamp">${ts}</span> ${msg}</div>` + "`" + `;
            }).join('');
            
            if (atBottom) logBox.scrollTop = logBox.scrollHeight;
            
            if (j.status === 'running') setTimeout(poll, 1000);
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
        <p style="text-align: center; color: var(--text-secondary); margin-bottom: 40px; font-size: 15px;">Search through indices discovered by the swarm</p>

        <div class="card" style="padding: 12px; border-radius: 100px; margin-bottom: 40px;">
            <div style="display: flex; gap: 12px; padding: 4px;">
                <input type="text" id="q" placeholder="Enter search keywords or neural intent..." onkeypress="if(event.key==='Enter') doSearch()" style="border-radius: 100px; padding: 16px 32px; background: transparent; border: none; font-size: 18px;">
                <button onclick="doSearch()" style="border-radius: 100px; padding: 0 40px;">Search</button>
            </div>
        </div>

        <div id="results-count" style="margin-bottom: 16px; margin-left: 8px; font-size: 14px; color: var(--text-secondary); display: none;"></div>
        <div id="results" class="card" style="padding: 0; display: none; overflow: hidden;"></div>
    </div>

    <script>
        document.getElementById('nav-discovery').classList.add('active');

        async function doSearch() {
            const q = document.getElementById('q').value;
            if(!q) return;
            
            const container = document.getElementById('results');
            const countLabel = document.getElementById('results-count');
            
            container.style.display = 'block';
            container.innerHTML = '<div style="padding: 80px; text-align: center;"><div style="color: var(--accent); margin-bottom: 16px; font-weight: 600;">ACCESSING DISTRIBUTED DATABASE...</div><div style="font-size: 13px; color: var(--text-secondary);">Querying memory nodes and disk indices</div></div>';
            
            try {
                const r = await fetch('/api/search?q=' + encodeURIComponent(q));
                const data = await r.json();
                container.innerHTML = '';
                
                if (!data || data.length === 0) {
                    countLabel.style.display = 'none';
                    container.innerHTML = ` + "`" + `
                        <div style="padding: 80px 40px; text-align: center;">
                            <div style="font-size: 48px; margin-bottom: 24px;">📡</div>
                            <div style="font-size: 22px; font-weight: 600; color: var(--text);">No matching signals found</div>
                            <div style="color: var(--text-secondary); margin-top: 12px; max-width: 400px; margin-left: auto; margin-right: auto;">The search query "${q}" did not return any results from the current index. Try reducing depth or expanding origin URLs.</div>
                        </div>
                    ` + "`" + `;
                    return;
                }
                
                data.sort((a,b) => b.relevance - a.relevance);
                countLabel.innerText = data.length + ' indexed pages relevant to your query';
                countLabel.style.display = 'block';

                data.forEach(res => {
                    const div = document.createElement('div');
                    div.className = 'search-result';
                    div.innerHTML = ` + "`" + `
                        <a href="${res.url}" target="_blank" class="search-result-url">${res.url}</a>
                        <div class="search-result-meta">
                            <div>Score: <b>${Math.round(res.relevance)}</b></div>
                            <div>Depth: <b>${res.depth}</b></div>
                            <div style="opacity: 0.6">Discovery Origin: <b>${new URL(res.origin_url).hostname}</b></div>
                        </div>
                    ` + "`" + `;
                    container.appendChild(div);
                });
            } catch (e) {
                container.innerHTML = '<div style="padding: 40px; text-align: center; color: var(--error)">DATABASE CONNECTION LOST: Unable to fetch neural index signals.</div>';
            }
        }
    </script>
</body></html>
`
