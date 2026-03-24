# Product Requirement Document (PRD): SpiderSearch

## 1. Goal
Build a localhost-runnable crawler and search system that demonstrates:
- `index(origin, k)` style crawling
- live `search(query)` while indexing is active
- visibility into queue depth and back-pressure state
- simple persistence on a single machine

## 2. Product Surface
- Crawler page: create jobs and review recent runs
- Status page: inspect logs, queue depth, worker activity, and back pressure
- Search page: query indexed pages by term
- Quiz answer document: a checked-in text artifact derived from real crawl data
- Fixture pages: deterministic local pages for repeatable crawling

## 3. Functional Requirements

### 3.1 Index
- Accept an `origin` URL and depth `k`
- Crawl up to `k` hops from the origin
- Never schedule the same URL twice
- Resolve both absolute and relative links
- Support back pressure with:
  - max concurrent workers
  - request hit-rate throttling
  - queue capacity
  - max URLs per job
- Persist job state and visited URLs to disk

### 3.2 Search
- Accept `query` and return relevant indexed URLs
- Return the relevant page URL with its `origin_url` and `depth`
- Work while indexing is still running
- Rank with a deterministic formula:

```text
(frequency * 10) + 1000 - (depth * 5)
```

### 3.3 Storage
- Store raw term entries in `data/storage/*.data`
- Use one file per leading character
- Store each record as:

```text
word url origin depth frequency
```

### 3.4 Observability
- Show per-job logs
- Show queue depth
- Show active workers
- Show whether back pressure is active

## 4. API Endpoints
- `GET /index?origin=<url>&k=<depth>`
- `GET /search?query=<text>&sortBy=relevance`
- `GET /api/status/<job_id>`
- `GET /api/jobs`

## 5. Constraints
- Go standard library first
- No external database required
- Localhost runnable
- Single-machine design, but structured for future migration to shared queues and distributed storage
