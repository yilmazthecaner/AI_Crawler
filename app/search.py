"""
Self-implemented TF-IDF search engine.

Scoring formula (per the spec):
    score = (term_frequency_in_title × 3 + term_frequency_in_body_text) / document_length

The final score for a document is the sum of per-term scores across all query terms.
Results are returned sorted by descending score.
"""

from __future__ import annotations

import re
import logging
from typing import Dict, List

from app import database as db

logger = logging.getLogger("search")

_WORD_RE = re.compile(r"[a-z0-9]+")


def _tokenize(text: str) -> List[str]:
    """Lowercase and split text into word tokens."""
    return _WORD_RE.findall(text.lower())


def _count_term(tokens: List[str], term: str) -> int:
    """Count exact occurrences of *term* in a token list."""
    return sum(1 for t in tokens if t == term)


async def search(query: str) -> List[Dict]:
    """
    Search indexed pages using self-implemented TF-IDF relevancy scoring.

    For each query term that appears in a page's title or body_text:
        term_score = (title_tf × 3 + body_tf) / doc_length

    Total page score = sum of term_scores for all query terms.
    """
    query_terms = _tokenize(query)
    if not query_terms:
        return []

    # Fetch candidate pages from the database
    rows = await db.search_pages(query_terms)
    if not rows:
        return []

    scored: List[Dict] = []

    for row in rows:
        title = row.get("title") or ""
        body = row.get("body_text") or ""

        title_tokens = _tokenize(title)
        body_tokens = _tokenize(body)

        # Document length is the total number of tokens
        doc_length = len(title_tokens) + len(body_tokens)
        if doc_length == 0:
            continue

        total_score = 0.0
        matched_terms = []

        for term in query_terms:
            title_tf = _count_term(title_tokens, term)
            body_tf = _count_term(body_tokens, term)

            if title_tf == 0 and body_tf == 0:
                continue

            matched_terms.append(term)
            term_score = (title_tf * 3 + body_tf) / doc_length
            total_score += term_score

        if total_score > 0:
            scored.append({
                "relevant_url": row["url"],
                "origin_url": row["origin_url"],
                "depth": row["depth"],
                "score": round(total_score, 6),
                "matched_terms": matched_terms,
            })

    # Sort by score descending, then by depth ascending as tiebreaker
    scored.sort(key=lambda x: (-x["score"], x["depth"]))

    return scored
