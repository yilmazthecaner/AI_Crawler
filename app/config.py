"""
Configuration module for the Multi-Agent Web Crawler.
All settings are configurable via environment variables with sensible defaults.
"""

import os


# ---------------------------------------------------------------------------
# Server
# ---------------------------------------------------------------------------
SERVER_HOST: str = os.getenv("SERVER_HOST", "0.0.0.0")
SERVER_PORT: int = int(os.getenv("SERVER_PORT", "8000"))

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------
DB_PATH: str = os.getenv("DB_PATH", "crawler.db")

# ---------------------------------------------------------------------------
# Crawler
# ---------------------------------------------------------------------------
CRAWL_RATE_PER_SEC: int = int(os.getenv("CRAWL_RATE_PER_SEC", "10"))
MAX_QUEUE_DEPTH: int = int(os.getenv("MAX_QUEUE_DEPTH", "500"))
QUEUE_RESUME_THRESHOLD: int = int(os.getenv("QUEUE_RESUME_THRESHOLD", "400"))
CRAWL_DELAY_PER_DOMAIN: float = float(os.getenv("CRAWL_DELAY_PER_DOMAIN", "0.5"))
STAY_ON_DOMAIN: bool = os.getenv("STAY_ON_DOMAIN", "true").lower() in ("true", "1", "yes")
MAX_CONCURRENT_WORKERS: int = int(os.getenv("MAX_CONCURRENT_WORKERS", "10"))
HTTP_TIMEOUT: int = int(os.getenv("HTTP_TIMEOUT", "15"))
MAX_RESPONSE_BYTES: int = int(os.getenv("MAX_RESPONSE_BYTES", str(5 * 1024 * 1024)))  # 5 MB
USER_AGENT: str = os.getenv("USER_AGENT", "MultiAgentCrawler/2.0 (+https://localhost)")

# Binary extensions to skip
BINARY_EXTENSIONS: set = {
    ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".ico",
    ".zip", ".tar", ".gz", ".rar", ".7z",
    ".mp3", ".mp4", ".avi", ".mov", ".wmv", ".flv",
    ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
    ".exe", ".dmg", ".bin", ".iso",
}
