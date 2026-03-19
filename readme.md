# SpiderSearch

SpiderSearch is a high-performance, concurrent web crawler and real-time search engine built in Go.

## Features
- **Concurrent Indexed Crawling**: Efficient recursion with thread-safe visited checks.
- **Back-Pressure Management**: Controlled worker pool via semaphores.
- **Real-Time Search**: Query the index while the crawler is still active.
- **Heuristic Ranking**: Results ranked by keyword frequency and title relevance.
- **Live Dashboard**: CLI monitor for processed vs. queued URLs and active workers.

## Architecture
- **Language**: Go 1.21+
- **Native Implementation**: Strictly uses `net/http`, `sync`, and `regexp` (No external libraries like Scrapy or BeautifulSoup).
- **Concurrency primitives**: Channels, Mutexes, and `sync.WaitGroup` for robust synchronization.

## Getting Started

### Prerequisites
- Go 1.21+ installed.

### Installation
```bash
go mod tidy
go build -o spidersearch .
```

### Usage
```bash
./spidersearch -origin https://go.dev -depth 2 -workers 10
```

### Search Prompt
Once the application starts, you can type queries directly:
```bash
SpiderSearch > golang
SpiderSearch > documentation
```
Type `exit` or `quit` to stop the session.

## Configuration Flags
- `-origin`: The starting URL for the crawl (default: `https://go.dev`).
- `-depth`: Max hops from the origin (default: `2`).
- `-workers`: Max concurrent crawl routines (default: `10`).
