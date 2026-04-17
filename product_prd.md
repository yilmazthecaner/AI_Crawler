# Product Requirements Document (PRD): Multi-Agent Web Crawler & Search System

## 1. Problem Statement

Build a multi-agent web crawler and search system that can:
- Accept a seed URL and crawl depth, then asynchronously index web pages using BFS
- Provide live, full-text search over indexed pages while crawling is still active
- Apply back-pressure controls to prevent overwhelming target servers and local resources
- Persist all state in SQLite so crawls survive server restarts
- Expose all functionality via a REST API and CLI tool

This is an upgrade from Project 1 (a Go-based single-machine crawler) into a Python-based system with stronger persistence, a standardized REST API, and multi-agent development documentation.

## 2. User Stories

1. **As a researcher**, I want to crawl a website by providing its URL and a depth limit, so that I can index all pages within that depth for later search.
2. **As a researcher**, I want to search across all indexed pages using keywords, so that I can find relevant content with TF-IDF scoring.
3. **As a system operator**, I want to monitor the crawler's queue depth, crawl rate, and back-pressure state in real time, so that I can observe system health.
4. **As a developer**, I want the crawl to resume automatically after a server restart, so that I don't lose progress on long-running crawls.
5. **As a user**, I want a CLI tool that can trigger crawling, search, and display live status, so that I can interact with the system from the terminal.

## 3. Functional Requirements

### 3.1 POST /index
- Accepts JSON: `{ "origin": "<url>", "k": <int> }`
- Starts an async background crawl from `origin` up to depth `k`
- Uses BFS; never crawls the same URL twice across the entire database
- Returns immediately with `{ "session_id": ..., "status": "started" }`
- Must be resumable: pending items survive server restarts

### 3.2 GET /search?q=\<query\>
- Accepts a query string parameter `q`
- Searches the `pages` table using self-implemented TF-IDF scoring:
  - `Score = (term_frequency_in_title × 3 + term_frequency_in_body_text) / document_length`
- Returns JSON array of `[{ "relevant_url", "origin_url", "depth" }]` sorted by relevance descending
- Reflects pages indexed so far, even during active crawling

### 3.3 GET /status
- Returns current system state:
```json
{
  "active_sessions": [...],
  "queue_depth": 42,
  "back_pressure_active": false,
  "pages_indexed": 150,
  "crawl_rate_per_sec": 8.5
}
```

### 3.4 Back-Pressure Controls
- **Max queue depth**: When pending queue exceeds 500 items, pause adding new URLs until it drains below 400
- **Rate limiting**: Crawl at most 10 pages/second (configurable via `CRAWL_RATE_PER_SEC` env var)
- **Per-domain delay**: Minimum 0.5 seconds between requests to the same domain

### 3.5 URL Filtering
- Only crawl HTTP/HTTPS URLs
- Skip binary files (.pdf, .jpg, .png, .zip, etc.)
- Stay on the origin's domain by default (configurable via `STAY_ON_DOMAIN` env var)
- Normalize URLs (strip fragments, resolve relative paths)

### 3.6 CLI
- `python cli.py index <url> <depth>` — triggers POST /index
- `python cli.py search <query>` — triggers GET /search, shows results table
- `python cli.py status` — polls GET /status every 2 seconds, shows live dashboard

## 4. Non-Functional Requirements

### 4.1 Scale
- Single-machine deployment using SQLite
- Support concurrent crawling of up to 10 pages simultaneously
- Handle databases with 10,000+ indexed pages

### 4.2 Back-Pressure
- Queue depth cap enforced at 500 pending items
- Rate limiter enforced at 10 requests/second
- Per-domain delay of 0.5 seconds minimum

### 4.3 Latency
- POST /index returns within 100ms (crawl runs in background)
- GET /search returns within 500ms for databases up to 10,000 pages
- GET /status returns within 100ms

### 4.4 Reliability
- Crawler is resumable across server restarts
- SQLite WAL mode enables concurrent read/write without blocking
- Graceful error handling — individual page failures don't crash the crawl session

## 5. Data Model

### pages
| Column | Type | Description |
|--------|------|-------------|
| url | TEXT PRIMARY KEY | The crawled page URL |
| origin_url | TEXT | The seed URL that initiated the crawl |
| depth | INTEGER | Link distance from origin |
| title | TEXT | Extracted `<title>` content |
| body_text | TEXT | Stripped text content |
| indexed_at | TIMESTAMP | When the page was indexed |

### crawl_queue
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PRIMARY KEY | Auto-increment ID |
| url | TEXT UNIQUE | URL to be crawled |
| origin_url | TEXT | Source crawl origin |
| depth | INTEGER | Target depth |
| status | TEXT | pending, processing, done, failed |
| added_at | TIMESTAMP | When the URL was enqueued |

### crawl_sessions
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PRIMARY KEY | Auto-increment session ID |
| origin_url | TEXT | Seed URL |
| max_depth | INTEGER | Maximum crawl depth |
| started_at | TIMESTAMP | Session start time |
| status | TEXT | running, finished |

## 6. API Contract

### POST /index
```
Request:  { "origin": "https://example.com", "k": 3 }
Response: { "session_id": 1, "status": "started" }
```

### GET /search?q=python
```
Response: [
  { "relevant_url": "https://example.com/python", "origin_url": "https://example.com", "depth": 1, "score": 0.045 },
  ...
]
```

### GET /status
```
Response: {
  "active_sessions": [{ "id": 1, "origin_url": "https://example.com", "max_depth": 3, "started_at": "...", "status": "running" }],
  "queue_depth": 42,
  "back_pressure_active": false,
  "pages_indexed": 150,
  "crawl_rate_per_sec": 8.5
}
```

## 7. Tech Stack Rationale

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Python 3.11+ | Excellent async ecosystem, academic readability |
| Web Framework | FastAPI | Auto-generates OpenAPI docs, native async, Pydantic validation |
| HTTP Client | aiohttp | Mature async HTTP with connection pooling |
| HTML Parser | BeautifulSoup4 | Industry standard, handles malformed HTML gracefully |
| Database | SQLite (aiosqlite) | Zero-config, WAL mode for concurrent access, file-portable |
| CLI | Rich | Beautiful terminal formatting with live display support |
| ASGI Server | Uvicorn | Production-grade ASGI server for FastAPI |
