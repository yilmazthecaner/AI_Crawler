"""
SQLite database layer using aiosqlite.

Tables
------
- pages          – every successfully crawled page
- crawl_queue    – BFS frontier (persistent across restarts)
- crawl_sessions – one row per POST /index invocation
"""

from __future__ import annotations

import aiosqlite
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple

from app import config

_db: Optional[aiosqlite.Connection] = None


# ── Lifecycle ───────────────────────────────────────────────────────────────

async def init_db() -> aiosqlite.Connection:
    """Open (or reuse) the database connection and create tables."""
    global _db
    if _db is not None:
        return _db

    _db = await aiosqlite.connect(config.DB_PATH)
    _db.row_factory = aiosqlite.Row

    # WAL mode for concurrent readers + single writer
    await _db.execute("PRAGMA journal_mode=WAL")
    await _db.execute("PRAGMA busy_timeout=5000")

    await _db.executescript("""
        CREATE TABLE IF NOT EXISTS pages (
            url         TEXT PRIMARY KEY,
            origin_url  TEXT NOT NULL,
            depth       INTEGER NOT NULL,
            title       TEXT,
            body_text   TEXT,
            indexed_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        CREATE TABLE IF NOT EXISTS crawl_queue (
            id          INTEGER PRIMARY KEY AUTOINCREMENT,
            url         TEXT UNIQUE NOT NULL,
            origin_url  TEXT NOT NULL,
            depth       INTEGER NOT NULL,
            status      TEXT NOT NULL DEFAULT 'pending',
            added_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        CREATE TABLE IF NOT EXISTS crawl_sessions (
            id          INTEGER PRIMARY KEY AUTOINCREMENT,
            origin_url  TEXT NOT NULL,
            max_depth   INTEGER NOT NULL,
            started_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            status      TEXT NOT NULL DEFAULT 'running'
        );

        CREATE INDEX IF NOT EXISTS idx_crawl_queue_status ON crawl_queue(status);
        CREATE INDEX IF NOT EXISTS idx_pages_origin ON pages(origin_url);
    """)
    await _db.commit()
    return _db


async def close_db() -> None:
    global _db
    if _db is not None:
        await _db.close()
        _db = None


async def get_db() -> aiosqlite.Connection:
    if _db is None:
        return await init_db()
    return _db


# ── Crawl Sessions ─────────────────────────────────────────────────────────

async def create_session(origin_url: str, max_depth: int) -> int:
    db = await get_db()
    cursor = await db.execute(
        "INSERT INTO crawl_sessions (origin_url, max_depth, started_at, status) VALUES (?, ?, ?, 'running')",
        (origin_url, max_depth, _now()),
    )
    await db.commit()
    return cursor.lastrowid  # type: ignore[return-value]


async def update_session_status(session_id: int, status: str) -> None:
    db = await get_db()
    await db.execute("UPDATE crawl_sessions SET status = ? WHERE id = ?", (status, session_id))
    await db.commit()


async def get_active_sessions() -> List[Dict[str, Any]]:
    db = await get_db()
    cursor = await db.execute(
        "SELECT id, origin_url, max_depth, started_at, status FROM crawl_sessions WHERE status = 'running'"
    )
    rows = await cursor.fetchall()
    return [dict(r) for r in rows]


async def get_all_sessions() -> List[Dict[str, Any]]:
    db = await get_db()
    cursor = await db.execute(
        "SELECT id, origin_url, max_depth, started_at, status FROM crawl_sessions ORDER BY started_at DESC"
    )
    rows = await cursor.fetchall()
    return [dict(r) for r in rows]


# ── Crawl Queue ─────────────────────────────────────────────────────────────

async def enqueue_url(url: str, origin_url: str, depth: int) -> bool:
    """Try to add a URL to the queue. Returns False if it already exists anywhere."""
    db = await get_db()
    try:
        await db.execute(
            "INSERT INTO crawl_queue (url, origin_url, depth, status, added_at) VALUES (?, ?, ?, 'pending', ?)",
            (url, origin_url, depth, _now()),
        )
        await db.commit()
        return True
    except aiosqlite.IntegrityError:
        return False


async def dequeue_url(url: str) -> None:
    """Mark a queue item as 'processing'."""
    db = await get_db()
    await db.execute("UPDATE crawl_queue SET status = 'processing' WHERE url = ?", (url,))
    await db.commit()


async def complete_url(url: str) -> None:
    """Mark a queue item as 'done'."""
    db = await get_db()
    await db.execute("UPDATE crawl_queue SET status = 'done' WHERE url = ?", (url,))
    await db.commit()


async def fail_url(url: str) -> None:
    """Mark a queue item as 'failed'."""
    db = await get_db()
    await db.execute("UPDATE crawl_queue SET status = 'failed' WHERE url = ?", (url,))
    await db.commit()


async def get_pending_count() -> int:
    db = await get_db()
    cursor = await db.execute("SELECT COUNT(*) FROM crawl_queue WHERE status = 'pending'")
    row = await cursor.fetchone()
    return row[0] if row else 0


async def get_pending_urls(limit: int = 50) -> List[Dict[str, Any]]:
    db = await get_db()
    cursor = await db.execute(
        "SELECT url, origin_url, depth FROM crawl_queue WHERE status = 'pending' ORDER BY id ASC LIMIT ?",
        (limit,),
    )
    rows = await cursor.fetchall()
    return [dict(r) for r in rows]


async def reset_processing_to_pending() -> int:
    """On startup, reset any 'processing' items back to 'pending' for resumability."""
    db = await get_db()
    cursor = await db.execute("UPDATE crawl_queue SET status = 'pending' WHERE status = 'processing'")
    await db.commit()
    return cursor.rowcount


# ── Pages ───────────────────────────────────────────────────────────────────

async def url_already_crawled(url: str) -> bool:
    db = await get_db()
    cursor = await db.execute("SELECT 1 FROM pages WHERE url = ?", (url,))
    return (await cursor.fetchone()) is not None


async def url_in_queue(url: str) -> bool:
    db = await get_db()
    cursor = await db.execute("SELECT 1 FROM crawl_queue WHERE url = ?", (url,))
    return (await cursor.fetchone()) is not None


async def insert_page(url: str, origin_url: str, depth: int, title: str, body_text: str) -> None:
    db = await get_db()
    await db.execute(
        """INSERT OR REPLACE INTO pages (url, origin_url, depth, title, body_text, indexed_at)
           VALUES (?, ?, ?, ?, ?, ?)""",
        (url, origin_url, depth, title, body_text, _now()),
    )
    await db.commit()


async def get_pages_count() -> int:
    db = await get_db()
    cursor = await db.execute("SELECT COUNT(*) FROM pages")
    row = await cursor.fetchone()
    return row[0] if row else 0


async def search_pages(query_terms: List[str]) -> List[Dict[str, Any]]:
    """
    Return all pages where *any* query term appears in title or body_text.
    Raw rows; scoring happens in the search module.
    """
    if not query_terms:
        return []

    db = await get_db()
    conditions = " OR ".join(
        ["(LOWER(title) LIKE ? OR LOWER(body_text) LIKE ?)" for _ in query_terms]
    )
    params: List[str] = []
    for term in query_terms:
        like = f"%{term.lower()}%"
        params.extend([like, like])

    cursor = await db.execute(
        f"SELECT url, origin_url, depth, title, body_text FROM pages WHERE {conditions}",
        params,
    )
    rows = await cursor.fetchall()
    return [dict(r) for r in rows]


# ── Helpers ─────────────────────────────────────────────────────────────────

def _now() -> str:
    return datetime.now(timezone.utc).isoformat()
