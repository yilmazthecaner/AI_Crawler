# SpiderSearch

SpiderSearch is a single-machine web crawler and live search demo written in Go. It indexes pages while exposing a filesystem-backed search API, a localhost UI, and a checked-in quiz answer document derived from real stored crawl data.

## What It Does
- Crawls from an `origin` URL up to depth `k`
- Avoids scheduling the same URL twice
- Applies back pressure with worker limits, hit-rate throttling, queue caps, and a max-URL ceiling
- Persists indexed terms to raw storage files under `data/storage/*.data`
- Serves search results while indexing is still running
- Exposes crawler, status, search, and fixture pages on localhost

## Local Run
```bash
go run main.go -port 3600
```

Open:
- `http://localhost:3600/`
- `http://localhost:3600/search`

## Deterministic Demo Crawl
The repo already includes generated crawl data in `data/storage/`, including `data/storage/p.data`.

If you want to regenerate it locally, start the app and crawl:

```text
origin = http://localhost:3600/fixture/start
k = 2
```

The local fixture pages are served by the app itself, so no external internet access is required.

## API
Create an index job:

```text
GET /index?origin=http://localhost:3600/fixture/start&k=2&hitRate=5&queueCap=50&maxURLs=20
```

Search:

```text
GET /search?query=python&sortBy=relevance
```

The search API returns:
- `url`
- `origin_url`
- `depth`
- `frequency`
- `relevance_score`

Scoring uses:

```text
(frequency * 10) + 1000 - (depth * 5)
```

## Raw Storage Format
Each raw storage file is line-based and follows:

```text
word url origin depth frequency
```

Example:

```text
python http://localhost:3600/fixture/python-basics http://localhost:3600/fixture/start 1 8
```

## UI Pages
- `/` creates crawl jobs and lists history
- `/status/<job_id>` shows queue depth, active workers, logs, and back-pressure state
- `/search` queries the live index
- `/raw/storage/p.data` opens the raw file directly

## Quiz Submission Text
The quiz answers are stored in:

`quiz_answers.md`
