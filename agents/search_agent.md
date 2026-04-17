# Search Agent

## Role Description

The Search Agent is responsible for implementing the search engine: full-text relevancy scoring over indexed pages using a self-implemented TF-IDF algorithm. It produces `app/search.py`.

## Prompt Given

```
You are the Search Agent. Implement a search engine in Python that:

1. Accepts a query string, tokenizes it into lowercase terms
2. Searches the SQLite `pages` table for pages where query terms appear in title or body_text
3. Implements TF-IDF relevancy scoring WITHOUT using any search library:
   - For each query term in each candidate page:
     score += (term_frequency_in_title × 3 + term_frequency_in_body_text) / document_length
   - document_length = total tokens in title + body_text
4. Returns results as [{ "relevant_url", "origin_url", "depth" }] sorted by score descending
5. Must work on live data — reads current DB state even during active crawling
6. Handles edge cases: empty queries, pages with no text, zero-length documents

Use only SQL + Python logic. No Elasticsearch, Whoosh, or similar libraries.
The search must be a pure async function that queries SQLite via the database module.
```

## Output Summary

The Search Agent produced `app/search.py` with the following implementation:

### Algorithm
1. **Tokenization**: `_tokenize(text)` uses regex `[a-z0-9]+` to extract lowercase word tokens
2. **Candidate retrieval**: SQL `LIKE` query across all query terms on title and body_text columns
3. **Per-term scoring**: For each candidate page and each query term:
   - Count occurrences in title tokens (`title_tf`)
   - Count occurrences in body tokens (`body_tf`)
   - `term_score = (title_tf * 3 + body_tf) / doc_length`
4. **Aggregation**: Sum all term_scores for each document
5. **Sorting**: Descending by total score, ascending by depth as tiebreaker

### Key Design Decisions
- Title weighting (×3) makes pages where terms appear in the title rank higher
- Document length normalization prevents long pages from dominating purely due to term count
- SQL-level pre-filtering reduces the number of pages that need Python-side scoring
- Matched terms are tracked and returned for potential UI display

### Scoring Formula
```
For each query term t in document d:
    term_score(t, d) = (count(t, title) × 3 + count(t, body)) / (len(title_tokens) + len(body_tokens))

total_score(d) = Σ term_score(t, d) for all query terms t
```

## Evaluation Notes

**Accepted:**
- Self-implemented scoring — no external search libraries used
- Title ×3 weighting matches the spec requirement
- Document length normalization is mathematically sound and prevents bias toward long pages
- SQL `LIKE` pre-filtering is pragmatic for SQLite scale; avoids loading entire pages table
- Graceful handling of empty documents (skip if doc_length == 0)

**Changed:**
- Initial version used raw term count without normalization; changed to normalize by document length per spec
- Added `matched_terms` tracking for potential UI display (not required by spec but useful for debugging)
- Changed from synchronous DB access to async (consistent with rest of codebase)
- Added tiebreaker sort by depth (ascending) when scores are equal — shallower pages are preferred
