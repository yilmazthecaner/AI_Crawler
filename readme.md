# SpiderSearch

SpiderSearch is a high-performance, concurrent web crawler and real-time search engine built in Go.

## Features
- **Deterministic Crawling**: Precise depth control (`k`) and hop tracing.
- **Advanced Back-Pressure**: Configurable hit rates (req/s), queue capacities, and worker limits.
- **Real-Time Search Discovery**: Search through discovered indices instantly while the swarm is still active.
- **Premium Glassmorphism UI**: High-end Web interface with real-time telemetry and log streaming.
- **Persistence & Resume**: Automatically saves state to disk, allowing sequences to be viewed or resumed after restart.

## Architecture
- **Language**: Go 1.21+
- **Native Implementation**: Strictly uses standard library components.
- **Storage**: Custom in-memory inverted index with segmented disk persistence (partitioned by characters).

## Getting Started

### Installation
```bash
go mod tidy
go run main.go
```

### Usage
1. Open `http://localhost:8080` in your browser.
2. Enter an **Origin URL** and **Depth**.
3. Configure **Back Pressure** settings (Hit Rate, Queue Cap).
4. Start the crawl and monitor live logs in the Status view.
5. Use the **Global Discovery** tab to search in real-time.
