# Multi-Agent Web Crawler & Search System

A Python-based multi-agent web crawler and search system with async BFS crawling, TF-IDF search, back-pressure controls, and SQLite persistence. Built as Project 2, upgrading the original Go-based SpiderSearch (Project 1, preserved in `project1/`).

## Features

- **Async BFS Crawler** — Concurrent page fetching with depth tracking
- **TF-IDF Search** — Self-implemented relevancy scoring with title ×3 weighting
- **Back-Pressure** — Queue depth cap (500) + rate limiting (10 pages/sec)
- **Resumability** — Pending crawl items survive server restarts
- **REST API** — FastAPI with POST /index, GET /search, GET /status
- **CLI** — Rich-based terminal tool with live status dashboard
- **SQLite WAL** — Concurrent read/write for search during active crawling

## Quick Start

### Prerequisites
- Python 3.11 or higher

### Install Dependencies

```bash
pip install -r requirements.txt
```

### Start the Server

```bash
python run.py
```

The server starts on `http://localhost:8000`. The database (`crawler.db`) is created automatically.

### Using the CLI

```bash
# Start a crawl
python cli.py index http://localhost:8000/fixture/start 2

# Search indexed pages
python cli.py search python

# Live status dashboard (Ctrl+C to stop)
python cli.py status
```

## API Reference

### POST /index — Start a Crawl

```bash
curl -X POST http://localhost:8000/index \
  -H "Content-Type: application/json" \
  -d '{"origin": "http://localhost:8000/fixture/start", "k": 2}'
```

Response:
```json
{"session_id": 1, "status": "started"}
```

### GET /search — Search Indexed Pages

```bash
curl "http://localhost:8000/search?q=python"
```

Response:
```json
[
  {
    "relevant_url": "http://localhost:8000/fixture/python-basics",
    "origin_url": "http://localhost:8000/fixture/start",
    "depth": 1,
    "score": 0.142857,
    "matched_terms": ["python"]
  }
]
```

### GET /status — System State

```bash
curl http://localhost:8000/status
```

Response:
```json
{
  "active_sessions": [],
  "queue_depth": 0,
  "back_pressure_active": false,
  "pages_indexed": 5,
  "crawl_rate_per_sec": 0.0
}
```

## Deterministic Demo Crawl

The server includes built-in fixture pages for testing:

```bash
# Start the server
python run.py

# In another terminal, start a crawl of the fixture pages
curl -X POST http://localhost:8000/index \
  -H "Content-Type: application/json" \
  -d '{"origin": "http://localhost:8000/fixture/start", "k": 2}'

# Wait a few seconds, then search
curl "http://localhost:8000/search?q=python"
```

The fixture pages are deterministic and require no internet access:
- `/fixture/start` — Entry point with links to 3 pages
- `/fixture/python-basics` — Heavy "python" term frequency
- `/fixture/program-patterns` — Design patterns content
- `/fixture/page-signals` — Search ranking signals
- `/fixture/pipeline-notes` — Deepest page (depth 2)

## Configuration

All settings are configurable via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8000 | Server port |
| `DB_PATH` | crawler.db | SQLite database path |
| `CRAWL_RATE_PER_SEC` | 10 | Max pages per second |
| `MAX_QUEUE_DEPTH` | 500 | Back-pressure trigger threshold |
| `QUEUE_RESUME_THRESHOLD` | 400 | Back-pressure release threshold |
| `CRAWL_DELAY_PER_DOMAIN` | 0.5 | Seconds between same-domain requests |
| `STAY_ON_DOMAIN` | true | Restrict crawl to origin domain |
| `MAX_CONCURRENT_WORKERS` | 10 | Parallel fetch workers |
| `HTTP_TIMEOUT` | 15 | HTTP request timeout (seconds) |

Example:
```bash
CRAWL_RATE_PER_SEC=20 MAX_QUEUE_DEPTH=1000 python run.py
```

## Resumability

The crawler is designed to survive server restarts:

1. All pending URLs are stored in the `crawl_queue` SQLite table
2. On startup, any items with `status='processing'` are reset to `'pending'`
3. Active sessions with `status='running'` are automatically resumed
4. Already-crawled pages (in the `pages` table) are never re-crawled

To test resumability:
```bash
# Start a crawl
python run.py &
curl -X POST http://localhost:8000/index \
  -H "Content-Type: application/json" \
  -d '{"origin": "https://example.com", "k": 3}'

# Kill the server mid-crawl
kill %1

# Restart — pending items will be picked up automatically
python run.py
```

## Search Scoring

The search engine uses self-implemented TF-IDF scoring:

```
For each query term in a page:
    term_score = (title_frequency × 3 + body_frequency) / document_length

total_score = sum of all term_scores
```

- **Title weighting (×3)** — Terms in the title count 3× more
- **Document length normalization** — Prevents long pages from dominating
- **Live results** — Search reflects pages indexed so far, even during active crawling

## Project Structure

```
AI_Crawler/
├── app/                      # Python application package
│   ├── main.py               # FastAPI endpoints + fixture pages
│   ├── crawler.py            # Async BFS crawler engine
│   ├── search.py             # TF-IDF search engine
│   ├── database.py           # SQLite database layer
│   ├── models.py             # Pydantic request/response models
│   └── config.py             # Environment-based configuration
├── cli.py                    # Rich-based CLI tool
├── run.py                    # Server entry point
├── requirements.txt          # Python dependencies
├── agents/                   # Multi-agent documentation (7 agents)
├── multi_agent_workflow.md   # Agent workflow documentation
├── product_prd.md            # Product Requirements Document
├── recommendation.md         # Production deployment recommendations
├── project1/                 # Original Go-based Project 1 (preserved)
└── crawler.db                # SQLite database (auto-created)
```

## Project 1 (Original Go Crawler)

The original Go-based SpiderSearch is preserved in the `project1/` directory. To run it:

```bash
cd project1
go run main.go -port 3600
```

Project 1 documentation: `project1/readme.md`, `project1/quiz_answers.md`
