"""
Async BFS Web Crawler with back-pressure controls.

Back-pressure mechanisms
------------------------
1. **Max queue depth** – when pending items exceed MAX_QUEUE_DEPTH (500),
   the crawler pauses adding new URLs until the queue drains below
   QUEUE_RESUME_THRESHOLD (400).
2. **Rate limiting** – at most CRAWL_RATE_PER_SEC pages/second via a
   token-bucket style asyncio limiter.
3. **Per-domain delay** – enforces CRAWL_DELAY_PER_DOMAIN seconds between
   consecutive requests to the same domain.

Resumability
------------
On server start, any crawl_queue items with status='pending' are picked up
and continued in a new background task.
"""

from __future__ import annotations

import asyncio
import logging
import re
import time
from collections import defaultdict
from typing import Dict, Optional, Set
from urllib.parse import urljoin, urlparse

import aiohttp
from bs4 import BeautifulSoup

from app import config, database as db

logger = logging.getLogger("crawler")

# ── Module State ────────────────────────────────────────────────────────────

_active_sessions: Dict[int, asyncio.Task] = {}
_back_pressure_active: bool = False
_crawl_count_window: list = []  # timestamps of recent crawls for rate calc
_domain_last_request: Dict[str, float] = defaultdict(float)
_lock = asyncio.Lock()

# Binary extension pattern
_BINARY_RE = re.compile(
    r"\.(?:" + "|".join(ext.lstrip(".") for ext in config.BINARY_EXTENSIONS) + r")$",
    re.IGNORECASE,
)


# ── Public API ──────────────────────────────────────────────────────────────

async def start_crawl(origin: str, max_depth: int) -> int:
    """Create a session, seed the queue, and launch the background crawl task."""
    session_id = await db.create_session(origin, max_depth)

    # Seed the origin URL into the queue (if not already crawled/queued)
    if not await db.url_already_crawled(origin) and not await db.url_in_queue(origin):
        await db.enqueue_url(origin, origin, 0)

    task = asyncio.create_task(_crawl_loop(session_id, origin, max_depth))
    _active_sessions[session_id] = task
    return session_id


async def resume_pending_crawls() -> int:
    """Called on startup to resume any interrupted crawl sessions."""
    # Reset items that were mid-processing when server stopped
    reset_count = await db.reset_processing_to_pending()
    if reset_count:
        logger.info("Reset %d processing items back to pending", reset_count)

    # Re-activate sessions that were still running
    active = await db.get_active_sessions()
    for session in active:
        sid = session["id"]
        task = asyncio.create_task(
            _crawl_loop(sid, session["origin_url"], session["max_depth"])
        )
        _active_sessions[sid] = task
        logger.info("Resumed crawl session %d for %s", sid, session["origin_url"])

    return len(active)


def is_back_pressure_active() -> bool:
    return _back_pressure_active


def get_crawl_rate() -> float:
    """Returns average crawl rate (pages/sec) over the last 10 seconds."""
    now = time.monotonic()
    window = [t for t in _crawl_count_window if now - t < 10.0]
    if not window:
        return 0.0
    return len(window) / min(10.0, now - window[0] + 0.001)


def get_active_session_ids() -> list:
    return list(_active_sessions.keys())


# ── Core Crawl Loop ────────────────────────────────────────────────────────

async def _crawl_loop(session_id: int, origin: str, max_depth: int) -> None:
    """
    Main crawl loop. Pulls pending URLs from the queue, processes them
    concurrently (up to MAX_CONCURRENT_WORKERS), and respects back-pressure.
    """
    global _back_pressure_active

    sem = asyncio.Semaphore(config.MAX_CONCURRENT_WORKERS)
    rate_sem = asyncio.Semaphore(config.CRAWL_RATE_PER_SEC)

    # Token refiller: replenishes rate_sem tokens every second
    async def _rate_refiller():
        while True:
            await asyncio.sleep(1.0)
            for _ in range(config.CRAWL_RATE_PER_SEC - rate_sem._value):
                rate_sem.release()

    refiller_task = asyncio.create_task(_rate_refiller())

    connector = aiohttp.TCPConnector(limit=config.MAX_CONCURRENT_WORKERS, ssl=False)
    timeout = aiohttp.ClientTimeout(total=config.HTTP_TIMEOUT)

    try:
        async with aiohttp.ClientSession(connector=connector, timeout=timeout) as session:
            tasks: Set[asyncio.Task] = set()
            idle_rounds = 0

            while True:
                # Fetch a batch of pending URLs
                pending = await db.get_pending_urls(limit=config.MAX_CONCURRENT_WORKERS)

                if not pending and not tasks:
                    idle_rounds += 1
                    if idle_rounds >= 3:
                        # No more work — finish session
                        break
                    await asyncio.sleep(1.0)
                    continue

                idle_rounds = 0

                for item in pending:
                    url = item["url"]
                    depth = item["depth"]
                    item_origin = item["origin_url"]

                    # Mark as processing
                    await db.dequeue_url(url)

                    # Wait for both concurrency and rate slots
                    await sem.acquire()
                    await rate_sem.acquire()

                    task = asyncio.create_task(
                        _crawl_page(session, url, item_origin, depth, max_depth, sem)
                    )
                    tasks.add(task)
                    task.add_done_callback(tasks.discard)

                # Brief yield to let crawl tasks run
                if tasks:
                    done, _ = await asyncio.wait(tasks, timeout=0.5)

    except asyncio.CancelledError:
        logger.info("Crawl session %d cancelled", session_id)
    except Exception as e:
        logger.exception("Crawl session %d failed: %s", session_id, e)
    finally:
        refiller_task.cancel()
        await db.update_session_status(session_id, "finished")
        _active_sessions.pop(session_id, None)
        logger.info("Crawl session %d finished", session_id)


async def _crawl_page(
    session: aiohttp.ClientSession,
    url: str,
    origin_url: str,
    depth: int,
    max_depth: int,
    sem: asyncio.Semaphore,
) -> None:
    """Fetch, parse, store a single page and discover child links."""
    global _back_pressure_active

    try:
        # Per-domain delay
        domain = urlparse(url).netloc
        async with _lock:
            last = _domain_last_request[domain]
            now = time.monotonic()
            wait = config.CRAWL_DELAY_PER_DOMAIN - (now - last)
            if wait > 0:
                await asyncio.sleep(wait)
            _domain_last_request[domain] = time.monotonic()

        # Fetch
        headers = {"User-Agent": config.USER_AGENT}
        async with session.get(url, headers=headers, allow_redirects=True) as resp:
            if resp.status < 200 or resp.status >= 300:
                logger.warning("HTTP %d for %s", resp.status, url)
                await db.fail_url(url)
                return

            content_type = resp.headers.get("Content-Type", "")
            if "text/html" not in content_type and "text/plain" not in content_type:
                await db.fail_url(url)
                return

            raw = await resp.read()
            if len(raw) > config.MAX_RESPONSE_BYTES:
                raw = raw[: config.MAX_RESPONSE_BYTES]
            html = raw.decode("utf-8", errors="replace")

        # Parse
        soup = BeautifulSoup(html, "html.parser")

        title = ""
        title_tag = soup.find("title")
        if title_tag:
            title = title_tag.get_text(strip=True)

        # Remove script/style then get text
        for tag in soup(["script", "style", "noscript"]):
            tag.decompose()
        body_text = soup.get_text(separator=" ", strip=True)

        # Store the page
        await db.insert_page(url, origin_url, depth, title, body_text)
        await db.complete_url(url)

        # Track crawl rate
        _crawl_count_window.append(time.monotonic())
        # Trim window to last 30 seconds
        cutoff = time.monotonic() - 30.0
        while _crawl_count_window and _crawl_count_window[0] < cutoff:
            _crawl_count_window.pop(0)

        logger.info("Crawled [depth=%d] %s", depth, url)

        # Discover child links (only if we haven't reached max depth)
        if depth < max_depth:
            await _discover_links(soup, url, origin_url, depth + 1)

    except asyncio.CancelledError:
        raise
    except Exception as e:
        logger.warning("Error crawling %s: %s", url, e)
        await db.fail_url(url)
    finally:
        sem.release()


async def _discover_links(
    soup: BeautifulSoup,
    base_url: str,
    origin_url: str,
    child_depth: int,
) -> None:
    """Extract links from a page and enqueue valid ones, respecting back-pressure."""
    global _back_pressure_active

    origin_domain = urlparse(origin_url).netloc
    seen_on_page: Set[str] = set()

    for tag in soup.find_all("a", href=True):
        href = tag["href"].strip()
        if not href or href.startswith("#") or href.startswith("javascript:"):
            continue

        resolved = urljoin(base_url, href)

        # Normalize: strip fragment
        parsed = urlparse(resolved)
        resolved = parsed._replace(fragment="").geturl()

        # Only HTTP/HTTPS
        if parsed.scheme not in ("http", "https"):
            continue

        # Skip binary files
        if _BINARY_RE.search(parsed.path):
            continue

        # Stay on domain (configurable)
        if config.STAY_ON_DOMAIN and parsed.netloc != origin_domain:
            continue

        # Deduplicate on this page
        if resolved in seen_on_page:
            continue
        seen_on_page.add(resolved)

        # Check back-pressure: queue depth cap
        pending_count = await db.get_pending_count()
        if pending_count >= config.MAX_QUEUE_DEPTH:
            _back_pressure_active = True
            logger.debug("Back-pressure active: queue depth %d >= %d", pending_count, config.MAX_QUEUE_DEPTH)
            # Wait for queue to drain
            while pending_count >= config.QUEUE_RESUME_THRESHOLD:
                await asyncio.sleep(0.5)
                pending_count = await db.get_pending_count()
            _back_pressure_active = False
            logger.debug("Back-pressure released: queue depth %d", pending_count)

        # Skip if already crawled or already queued
        if await db.url_already_crawled(resolved):
            continue
        if await db.url_in_queue(resolved):
            continue

        await db.enqueue_url(resolved, origin_url, child_depth)
