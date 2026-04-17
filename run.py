#!/usr/bin/env python3
"""
Server entry point.
Starts the FastAPI application on the configured host and port.
"""

import uvicorn
from app.config import SERVER_HOST, SERVER_PORT


def main():
    uvicorn.run(
        "app.main:app",
        host=SERVER_HOST,
        port=SERVER_PORT,
        reload=False,
        log_level="info",
    )


if __name__ == "__main__":
    main()
