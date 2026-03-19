# Product Requirement Document (PRD): SpiderSearch

## 1. Project Overview
SpiderSearch is a high-performance, single-machine web crawler and real-time search engine. It aims to demonstrate efficient concurrency management, architectural sensibility, and real-time data processing without relying on high-level libraries.

## 2. Core Functionalities

### 2.1 The Indexer (Crawler)
- **Recursive Depth Crawling**: Crawls from an `origin` URL up to `k` hops.
- **Uniqueness Guarantee**: Implements a thread-safe "Visited" set to avoid redundant work and cycles.
- **Back-Pressure Management**: Limits concurrent workers and manages queue depth to avoid resource exhaustion.
- **Raw Implementation**: Uses only native `net/http` and `net/url` (Go standard library).

### 2.2 The Searcher
- **Live Querying**: Search engine functional *while* crawling is in progress.
- **Result Triples**: Returns `(relevant_url, origin_url, depth)`.
- **Thread Safety**: Uses concurrent-safe data structures (Mutexes, Channels) to allow simultaneous Read/Write.
- **Relevancy Ranking**: Heuristic-based ranking (e.g., term frequency, title matches).

### 2.3 System Visibility
- **Dashboard**: Real-time CLI or Web UI tracking:
    - Progress (Processed vs. Queued URLs)
    - Queue Depth
    - Throttling/Back-pressure status

### 2.4 Persistence
- **Pause/Resume**: Capability to save state and resume indexing after interruption.

## 3. Technical Constraints
- **Language**: Go (Golang) for its robust concurrency primitives.
- **No Heavy Libraries**: No Scrapy, BeautifulSoup, or external databases.
- **Single Machine**: Optimized for performance on a single host.

## 4. Success Criteria
- Successful recursive crawl to depth `k`.
- Search results reflect newly crawled pages within seconds.
- System handles 100+ concurrent requests/crawls without crashing.
