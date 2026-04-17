# Architect Agent

## Role Description

The Architect Agent is responsible for designing the overall system architecture for the Multi-Agent Web Crawler & Search System. This includes defining component boundaries, data flow patterns, the concurrency model, the back-pressure strategy, and the technology stack selection.

## Prompt Given

```
You are the Architect Agent. Design a multi-agent web crawler and search system that upgrades
an existing Go-based single-machine crawler (Project 1) into a Python-based system with:

1. A REST API (FastAPI) serving POST /index, GET /search, GET /status
2. SQLite as the local database with tables: pages, crawl_queue, crawl_sessions
3. Async BFS crawling with back-pressure (queue depth cap + rate limiting)
4. TF-IDF search that works concurrently with active indexing
5. A CLI for triggering index/search and observing live system state

Constraints:
- All data in SQLite (localhost), no external search engines
- Async/concurrent crawling required (asyncio)
- Persistent URL deduplication (survives restarts)
- Configurable crawl delay per domain (default 0.5s)
- Stay-on-domain crawling by default

Deliver: component diagram, data flow, concurrency model, back-pressure strategy, tech stack.
```

## Output Summary

The Architect Agent produced the following system design:

### Component Boundaries
- **app/config.py** — Centralized environment-based configuration
- **app/database.py** — Async SQLite layer (aiosqlite) with WAL mode for concurrent reads
- **app/crawler.py** — BFS crawl engine with back-pressure and rate limiting
- **app/search.py** — Self-implemented TF-IDF scoring engine
- **app/main.py** — FastAPI REST API with lifespan hooks
- **app/models.py** — Pydantic request/response schemas
- **cli.py** — Rich-based CLI client
- **run.py** — Server entry point

### Data Flow
```
User → POST /index → FastAPI → crawler.start_crawl()
                                  ↓
                            SQLite crawl_queue (BFS frontier)
                                  ↓
                            _crawl_loop() → aiohttp fetch → BeautifulSoup parse
                                  ↓
                            SQLite pages table
                                  ↓
User → GET /search → FastAPI → search.search() → reads pages table → scored results
```

### Concurrency Model
- asyncio event loop for all I/O
- asyncio.Semaphore for worker concurrency cap (10 workers)
- Token-bucket refiller for rate limiting (10 pages/sec)
- Per-domain delay tracking via monotonic timestamp map

### Back-Pressure Strategy
1. **Queue depth cap**: When pending queue exceeds 500 items, link discovery pauses until it drains below 400 (hysteresis to prevent oscillation)
2. **Rate limiting**: Token bucket refills at CRAWL_RATE_PER_SEC tokens/second; each page fetch consumes one token

### Technology Stack
- Python 3.11+ with asyncio
- FastAPI + Uvicorn (ASGI server)
- aiohttp (async HTTP client)
- aiosqlite (async SQLite with WAL)
- BeautifulSoup4 (HTML parsing)
- Rich (CLI terminal UI)
- httpx (CLI HTTP client)

## Evaluation Notes

**Accepted:**
- SQLite with WAL mode — enables concurrent read/write without locks blocking search during indexing
- Hysteresis in back-pressure (500 trigger, 400 resume) — prevents rapid on/off cycling
- Separation of `crawl_queue` from `pages` table — clean BFS frontier vs. indexed content
- Environment-variable configuration — makes all parameters tunable without code changes
- Pydantic models for API — auto-generates OpenAPI docs and validates input

**Changed:**
- Initially proposed Redis for the queue; changed to SQLite to meet the constraint of no external dependencies
- Initially proposed a separate process for the crawler; changed to in-process asyncio tasks for simplicity since this is a single-machine academic project
- Added `crawl_sessions` table (not in initial design) to track individual crawl invocations for the /status endpoint
