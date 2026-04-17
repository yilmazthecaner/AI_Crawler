# UI/CLI Agent

## Role Description

The UI/CLI Agent is responsible for building the command-line interface that allows users to trigger index/search operations and view live system state. It produces `cli.py`.

## Prompt Given

```
You are the UI/CLI Agent. Build a Python CLI tool using the `rich` library that provides:

1. `python cli.py index <url> <depth>`
   - Sends POST /index to the API server
   - Pretty-prints the session ID and status

2. `python cli.py search <query>`
   - Sends GET /search?q=<query> to the API server
   - Displays results as a formatted table with columns: #, Relevant URL, Origin URL, Depth, Score

3. `python cli.py status`
   - Polls GET /status every 2 seconds
   - Displays a live-updating dashboard with:
     - Pages indexed count
     - Queue depth
     - Back-pressure status (active/inactive)
     - Crawl rate (pages/sec)
     - List of active sessions

Use `rich.live.Live` for the live status view, `rich.table.Table` for search results,
and `rich.panel.Panel` for formatted output. Use `httpx` for HTTP requests.
Show helpful error messages when the server is unreachable.
```

## Output Summary

### Commands Implemented

| Command | Description |
|---------|-------------|
| `index <url> <depth>` | POST to /index, display session panel |
| `search <query>` | GET /search, display results table |
| `status` | Live polling dashboard with Rich Live |

### Library Usage
- **`rich.console.Console`** — Main output rendering
- **`rich.table.Table`** — Search results with styled columns
- **`rich.live.Live`** — Auto-refreshing status dashboard
- **`rich.panel.Panel`** — Bordered output boxes for index response and status
- **`httpx`** — Synchronous HTTP client for CLI commands

### Status Dashboard Features
- Updates every 2 seconds via `Live` context manager
- Shows: pages indexed, queue depth, back-pressure indicator, crawl rate, active sessions
- Graceful Ctrl+C handling to stop the polling loop
- Connection error display when server is unreachable

### Error Handling
- Connection errors show clear "Is the server running?" message
- Usage errors show command syntax with Rich formatting
- Unknown commands produce helpful feedback

## Evaluation Notes

**Accepted:**
- Rich library provides excellent terminal formatting without heavy dependencies
- Live dashboard with 2-second polling matches spec requirement
- httpx for synchronous HTTP — simpler than async for a CLI tool
- Panel formatting for index responses gives clear visual feedback
- Table formatting for search results is highly readable

**Changed:**
- Initial version used `argparse`; simplified to raw `sys.argv` parsing since there are only 3 commands
- Added multi-word search support: `sys.argv[2:]` joined for search queries
- Added score column to search results table (beyond spec minimum) for user insight
- Changed status refresh rate from 1/sec to 0.5/sec in Rich Live, while actual data polling stays at 2-second intervals to reduce API load
