"""
FastAPI application — REST API + fixture pages.

Endpoints
---------
- POST /index         — start a crawl session
- GET  /search?q=...  — search indexed pages
- GET  /status        — system status dashboard
- GET  /fixture/...   — deterministic test pages
"""

from __future__ import annotations

import asyncio
import logging
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, Query, Request
from fastapi.responses import HTMLResponse, JSONResponse

from app import config, database as db, crawler, search
from app.models import (
    IndexRequest,
    IndexResponse,
    SearchResultItem,
    StatusResponse,
    SessionInfo,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(name)s] %(levelname)s: %(message)s",
)
logger = logging.getLogger("api")


# ── Lifespan ────────────────────────────────────────────────────────────────

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup / shutdown hooks."""
    await db.init_db()
    logger.info("Database initialized (%s)", config.DB_PATH)

    # Resume any pending crawls from a previous run
    resumed = await crawler.resume_pending_crawls()
    if resumed:
        logger.info("Resumed %d pending crawl session(s)", resumed)

    yield

    await db.close_db()
    logger.info("Database connection closed")


app = FastAPI(
    title="Multi-Agent Web Crawler & Search",
    version="2.0.0",
    lifespan=lifespan,
)


# ── POST /index ─────────────────────────────────────────────────────────────

@app.post("/index", response_model=IndexResponse)
async def index_endpoint(payload: IndexRequest):
    """
    Start an async background crawl from `origin` up to depth `k`.
    Returns immediately with a session_id.
    """
    origin = payload.origin.strip()
    depth = payload.k

    logger.info("POST /index — origin=%s depth=%d", origin, depth)

    session_id = await crawler.start_crawl(origin, depth)
    return IndexResponse(session_id=session_id, status="started")


# ── GET /search ─────────────────────────────────────────────────────────────

@app.get("/search")
async def search_endpoint(q: str = Query(..., min_length=1, description="Search query")):
    """
    Search indexed pages using TF-IDF relevancy scoring.
    Returns results even if indexing is still in progress.
    """
    logger.info("GET /search — q=%s", q)

    results = await search.search(q)
    return results


# ── GET /status ─────────────────────────────────────────────────────────────

@app.get("/status", response_model=StatusResponse)
async def status_endpoint():
    """Return current system state."""
    active = await db.get_active_sessions()
    queue_depth = await db.get_pending_count()
    pages_indexed = await db.get_pages_count()
    rate = crawler.get_crawl_rate()

    sessions = [
        SessionInfo(
            id=s["id"],
            origin_url=s["origin_url"],
            max_depth=s["max_depth"],
            started_at=str(s["started_at"]),
            status=s["status"],
        )
        for s in active
    ]

    return StatusResponse(
        active_sessions=sessions,
        queue_depth=queue_depth,
        back_pressure_active=crawler.is_back_pressure_active(),
        pages_indexed=pages_indexed,
        crawl_rate_per_sec=round(rate, 2),
    )


# ── Fixture Pages ───────────────────────────────────────────────────────────
# Deterministic local pages for testing (carried forward from Project 1).

FIXTURE_PAGES = {
    "start": {
        "title": "Python Crawl Start",
        "body": (
            "<p>Python explorers use this page to test the crawler. "
            "This python page links to python practice notes, program patterns, "
            "and page ranking signals.</p>"
            "<p>Every page in this mini site is local, deterministic, and safe "
            "to crawl during the assignment review.</p>"
        ),
        "links": ["python-basics", "program-patterns", "page-signals"],
    },
    "python-basics": {
        "title": "Python Basics Page",
        "body": (
            "<p>Python learners start here. Python syntax, python variables, "
            "python functions, and python loops appear together on this page so "
            "the crawler can store repeated python terms.</p>"
            "<p>This page also references practical program structure and page scoring.</p>"
        ),
        "links": ["page-signals", "pipeline-notes"],
    },
    "program-patterns": {
        "title": "Program Patterns For Python",
        "body": (
            "<p>Program design patterns help python services stay predictable. "
            "This page mentions python pipelines, python indexing, and program "
            "planning for production style crawler work.</p>"
            "<p>Practice pages like this keep the sample crawl data easy to inspect.</p>"
        ),
        "links": ["pipeline-notes"],
    },
    "page-signals": {
        "title": "Page Signals For Python Search",
        "body": (
            "<p>Page ranking signals are intentionally simple here. Python appears "
            "repeatedly because python frequency should clearly influence the "
            "relevance score, while page depth adds a small penalty.</p>"
            "<p>This page talks about python recall, python precision, "
            "and python search explainability.</p>"
        ),
        "links": ["pipeline-notes"],
    },
    "pipeline-notes": {
        "title": "Pipeline Notes",
        "body": (
            "<p>Pipeline notes describe how a crawler can parse pages, persist "
            "terms, and publish search results while indexing is still active.</p>"
            "<p>Python is mentioned again here so deeper pages still contribute "
            "useful quiz data.</p>"
        ),
        "links": [],
    },
}


@app.get("/fixture/{page_name}", response_class=HTMLResponse)
async def fixture_page(page_name: str, request: Request):
    """Serve deterministic fixture pages for testing."""
    page = FIXTURE_PAGES.get(page_name)
    if not page:
        return HTMLResponse("<h1>404 Not Found</h1>", status_code=404)

    base = str(request.base_url).rstrip("/")
    links_html = "\n".join(
        f'<li><a href="{base}/fixture/{link}">{base}/fixture/{link}</a></li>'
        for link in page["links"]
    )

    html = f"""<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{page["title"]}</title>
</head>
<body>
    <main>
        <h1>{page["title"]}</h1>
        {page["body"]}
        <h2>Links</h2>
        <ul>{links_html}</ul>
    </main>
</body>
</html>"""
    return HTMLResponse(html)
