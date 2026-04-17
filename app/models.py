"""
Pydantic models for API request / response shapes.
"""

from __future__ import annotations

from typing import List, Optional
from pydantic import BaseModel, Field


# ── Requests ────────────────────────────────────────────────────────────────

class IndexRequest(BaseModel):
    origin: str = Field(..., description="The seed URL to start crawling from")
    k: int = Field(..., ge=1, le=50, description="Maximum crawl depth (hops from origin)")


# ── Responses ───────────────────────────────────────────────────────────────

class IndexResponse(BaseModel):
    session_id: int
    status: str = "started"


class SearchResultItem(BaseModel):
    relevant_url: str
    origin_url: str
    depth: int
    score: Optional[float] = None


class SearchResponse(BaseModel):
    query: str
    results: List[SearchResultItem]
    total: int


class SessionInfo(BaseModel):
    id: int
    origin_url: str
    max_depth: int
    started_at: str
    status: str


class StatusResponse(BaseModel):
    active_sessions: List[SessionInfo]
    queue_depth: int
    back_pressure_active: bool
    pages_indexed: int
    crawl_rate_per_sec: float
