# API Agent

## Role Description

The API Agent is responsible for wrapping the indexer and search components into a clean REST API using FastAPI. It produces `app/main.py` and `app/models.py`.

## Prompt Given

```
You are the API Agent. Build a FastAPI REST API that exposes the following endpoints:

1. POST /index
   - Accepts JSON: { "origin": "<url>", "k": <int> }
   - Starts an async background crawl via the crawler module
   - Returns immediately: { "session_id": ..., "status": "started" }

2. GET /search?q=<query>
   - Calls the search module with the query
   - Returns JSON list of { "relevant_url", "origin_url", "depth" } sorted by relevance

3. GET /status
   - Returns system state:
     {
       "active_sessions": [...],
       "queue_depth": <int>,
       "back_pressure_active": <bool>,
       "pages_indexed": <int>,
       "crawl_rate_per_sec": <float>
     }

Also:
- Use FastAPI lifespan for DB init/close and crawl resumption on startup
- Serve /fixture/{page_name} endpoints for deterministic test pages
- Use Pydantic models for all request/response shapes
- Handle validation errors gracefully

The server should run on port 8000 via Uvicorn.
```

## Output Summary

### Endpoints Implemented

| Method | Path | Description |
|--------|------|-------------|
| POST | `/index` | Start crawl session, returns session_id |
| GET | `/search?q=...` | TF-IDF search over indexed pages |
| GET | `/status` | System state dashboard |
| GET | `/fixture/{page_name}` | Deterministic test pages |

### Pydantic Models (`app/models.py`)
- `IndexRequest` — validated input with origin (str) and k (int, 1-50)
- `IndexResponse` — session_id + status
- `SearchResultItem` — relevant_url, origin_url, depth, score
- `StatusResponse` — full system state with typed fields
- `SessionInfo` — individual session metadata

### Lifespan Hooks
- **Startup**: Initialize SQLite database, resume pending crawls
- **Shutdown**: Close database connection

### Fixture Pages
Five deterministic pages ported from Project 1's Go fixtures:
- `start` → `python-basics`, `program-patterns`, `page-signals`
- `python-basics` → `page-signals`, `pipeline-notes`
- `program-patterns` → `pipeline-notes`
- `page-signals` → `pipeline-notes`
- `pipeline-notes` → (no outgoing links)

## Evaluation Notes

**Accepted:**
- FastAPI with Pydantic — provides auto-validation, OpenAPI docs, and type safety
- Lifespan context manager — clean startup/shutdown lifecycle
- Fixture pages ported from Project 1 — ensures backward compatibility for testing
- POST for /index (not GET) — semantically correct since it creates a resource

**Changed:**
- Initial version had /index as GET (matching Project 1); changed to POST per spec requirement with JSON body
- Added Pydantic field validation: depth capped at 1-50, origin is required string
- Added structured logging with timestamps for better observability
- Fixture pages initially returned plain text; changed to full HTML documents for proper crawling
