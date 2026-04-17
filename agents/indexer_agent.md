# Indexer Agent

## Role Description

The Indexer Agent is responsible for implementing the crawl engine: the BFS traversal, async work queue, depth tracking, URL deduplication, and back-pressure mechanisms (rate limiting + max queue depth). It produces `app/crawler.py`.

## Prompt Given

```
You are the Indexer Agent. Implement an async BFS web crawler in Python that:

1. Accepts a seed URL and max depth, then crawls pages breadth-first
2. Uses aiohttp for HTTP requests and BeautifulSoup for HTML parsing
3. Stores crawled pages in a SQLite `pages` table via the database module
4. Manages a persistent BFS frontier via the `crawl_queue` table
5. Never crawls the same URL twice (check both `pages` and `crawl_queue` tables)
6. Implements back-pressure:
   a. Max queue depth: pause link discovery when pending > 500, resume when < 400
   b. Rate limiting: max 10 pages/second via token bucket (configurable env var)
   c. Per-domain delay: 0.5s between requests to same domain
7. Filters URLs: HTTP/HTTPS only, no binary files, stay-on-domain by default
8. Is resumable: on startup, pick up pending queue items and continue crawling
9. Tracks crawl rate (pages/sec) for the status endpoint

Use asyncio.Semaphore for concurrency control, monotonic timestamps for per-domain delays.
Expose public functions: start_crawl(), resume_pending_crawls(), is_back_pressure_active(),
get_crawl_rate(), get_active_session_ids().
```

## Output Summary

The Indexer Agent produced `app/crawler.py` with the following implementation:

### Core Functions
- `start_crawl(origin, max_depth)` — Creates a crawl session, seeds the queue, launches `_crawl_loop` as an asyncio task
- `resume_pending_crawls()` — Called on startup; resets 'processing' → 'pending', re-launches tasks for active sessions
- `_crawl_loop(session_id, origin, max_depth)` — Main loop: fetches pending URLs, dispatches to `_crawl_page()` workers
- `_crawl_page(session, url, ...)` — Fetches a single page, parses HTML, stores in DB, discovers child links
- `_discover_links(soup, base_url, ...)` — Extracts href links, filters, checks back-pressure, enqueues valid URLs

### Back-Pressure Implementation
1. **Queue depth cap**: In `_discover_links()`, checks `get_pending_count()` before each enqueue. When count ≥ 500, enters a polling sleep loop until count drops below 400.
2. **Rate limiting**: Token-bucket via `asyncio.Semaphore(CRAWL_RATE_PER_SEC)` + a refiller coroutine that releases tokens every second.
3. **Concurrency cap**: `asyncio.Semaphore(MAX_CONCURRENT_WORKERS)` limits parallel fetch operations.
4. **Per-domain delay**: Tracks `_domain_last_request[domain]` timestamps; sleeps if interval < 0.5s.

### URL Filtering
- Strips fragment from URLs
- Rejects non-HTTP/HTTPS schemes
- Rejects binary extensions via regex pattern
- Enforces stay-on-domain by comparing netloc against origin domain
- Deduplicates on-page (set) and cross-crawl (DB check)

### Resumability
- `reset_processing_to_pending()` on startup recovers items that were mid-processing when server stopped
- Active sessions with `status='running'` get new asyncio tasks created for them

## Evaluation Notes

**Accepted:**
- Token bucket pattern for rate limiting — clean and precise
- Hysteresis thresholds for queue depth back-pressure — prevents oscillation
- WAL-mode SQLite enables search queries to read while crawler writes
- Per-domain delay prevents hammering individual servers
- Binary extension filter is comprehensive

**Changed:**
- Initial version used `asyncio.Queue` (in-memory) for BFS frontier; changed to SQLite `crawl_queue` table for persistence across restarts
- Added the `_lock` asyncio.Lock around per-domain delay logic to prevent race conditions in timestamp updates
- Increased idle round threshold from 1 to 3 before declaring session finished, preventing premature termination on slow-draining queues
- Added content-type check to skip non-HTML responses gracefully instead of crashing on binary content
