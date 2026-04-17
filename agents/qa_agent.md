# QA Agent

## Role Description

The QA Agent reviews all code produced by the other agents for correctness, edge cases, and architectural quality. It produces a critique with suggested fixes and validates that the system meets all requirements.

## Prompt Given

```
You are the QA Agent. Review the complete codebase of the Multi-Agent Web Crawler & Search
System for:

1. Correctness — Does each endpoint behave per spec? Are edge cases handled?
2. Concurrency safety — Are there race conditions in shared state?
3. Back-pressure — Is it truly enforced? Can it be bypassed?
4. Resumability — Will pending items actually be picked up on restart?
5. Search accuracy — Does the TF-IDF scoring match the specified formula?
6. URL handling — Deduplication, normalization, binary filtering, domain restriction
7. Error handling — Graceful failures, no crash loops, connection timeouts
8. Code quality — Consistent style, proper logging, clear separation of concerns

Produce: a categorized list of issues found, severity level, and suggested fixes.
```

## Output Summary

### Issues Found and Resolutions

#### Critical (Fixed)
1. **Race condition in per-domain delay** — Multiple coroutines could read/write `_domain_last_request` simultaneously.
   - *Fix*: Added `asyncio.Lock` around domain delay logic in `_crawl_page()`

2. **Rate limiter token leak** — `asyncio.Semaphore` refiller could over-release tokens if value already at max.
   - *Fix*: Refiller only releases `CRAWL_RATE_PER_SEC - rate_sem._value` tokens (capped to actual deficit)

3. **Premature session termination** — If queue drain was slow, crawler would exit after one idle round before pending items could be fetched.
   - *Fix*: Increased idle_rounds threshold to 3 with 1-second sleep between checks

#### High (Fixed)
4. **Content-type check missing** — Crawler would attempt to parse binary responses (images, PDFs served without proper extension).
   - *Fix*: Added `Content-Type` header check for `text/html` or `text/plain`

5. **SQLite busy errors under load** — Without WAL mode, concurrent read and write operations could deadlock.
   - *Fix*: Enabled `PRAGMA journal_mode=WAL` and `PRAGMA busy_timeout=5000`

6. **Queue dedup gap** — Between checking `url_already_crawled()` and `url_in_queue()`, another coroutine could enqueue the same URL.
   - *Fix*: `crawl_queue.url` has a UNIQUE constraint, so `IntegrityError` on duplicate insert is caught gracefully

#### Medium (Fixed)
7. **Crawl rate window unbounded growth** — `_crawl_count_window` list would grow indefinitely during long crawls.
   - *Fix*: Trim entries older than 30 seconds after each append

8. **Missing URL normalization** — Fragment-only URLs (#section) were not being stripped.
   - *Fix*: `urlparse(resolved)._replace(fragment="").geturl()` strips fragments

9. **Large response body memory usage** — Unlimited response bodies could cause OOM.
   - *Fix*: `MAX_RESPONSE_BYTES` cap (5 MB) applied during read

#### Low (Noted)
10. **No robots.txt compliance** — Crawler does not check robots.txt. Acceptable for academic project; flagged in recommendation.md for production.

11. **No retry on transient failures** — Failed URLs are marked as 'failed' without retry. Could add exponential backoff for production.

12. **SQL LIKE performance** — Full-text search via `LIKE '%term%'` is O(n) scan. Acceptable for academic scale; production should use FTS5 or Elasticsearch.

### Architectural Quality Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| Separation of concerns | ✅ Excellent | Clean module boundaries |
| Error handling | ✅ Good | Graceful failures, no crash loops |
| Configurability | ✅ Excellent | All parameters via env vars |
| Logging | ✅ Good | Structured with module names |
| Type safety | ✅ Good | Pydantic models + type hints |
| Testability | ⚠️ Fair | No unit tests; async code is testable via pytest-asyncio |

## Evaluation Notes

**Accepted:**
- All critical and high-severity issues were fixed before final code submission
- Architectural feedback on module boundaries was incorporated from the start (Architect Agent's design)
- SQLite WAL mode recommendation was adopted immediately

**Changed:**
- Added the asyncio.Lock for domain delay (identified by QA)
- Added content-type checking (identified by QA)
- Added crawl rate window trimming (identified by QA)
- Increased idle rounds from 1→3 (identified by QA)
- QA recommended adding unit tests; deferred for follow-up due to academic project scope
