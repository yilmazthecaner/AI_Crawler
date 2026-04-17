#!/usr/bin/env python3
"""
CLI tool for the Multi-Agent Web Crawler & Search System.

Commands
--------
    python cli.py index <url> <depth>   — Start a crawl and show session ID
    python cli.py search <query>        — Search and pretty-print results
    python cli.py status                — Live dashboard (polls every 2s)
"""

import sys
import time
import json

import httpx
from rich.console import Console
from rich.table import Table
from rich.live import Live
from rich.panel import Panel
from rich.layout import Layout
from rich.text import Text

API_BASE = "http://localhost:8000"
console = Console()


# ── index ───────────────────────────────────────────────────────────────────

def cmd_index(url: str, depth: int) -> None:
    """Trigger POST /index and display the session ID."""
    console.print(f"\n[bold cyan]🕷️  Starting crawl...[/bold cyan]")
    console.print(f"   Origin: [green]{url}[/green]")
    console.print(f"   Depth:  [green]{depth}[/green]\n")

    try:
        resp = httpx.post(
            f"{API_BASE}/index",
            json={"origin": url, "k": depth},
            timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()

        console.print(Panel.fit(
            f"[bold green]✓ Crawl started[/bold green]\n"
            f"  Session ID: [bold]{data['session_id']}[/bold]\n"
            f"  Status:     {data['status']}",
            title="Index Response",
            border_style="green",
        ))
    except httpx.ConnectError:
        console.print("[bold red]✗ Cannot connect to server. Is it running on :8000?[/bold red]")
    except Exception as e:
        console.print(f"[bold red]✗ Error: {e}[/bold red]")


# ── search ──────────────────────────────────────────────────────────────────

def cmd_search(query: str) -> None:
    """Trigger GET /search and display results as a table."""
    console.print(f"\n[bold cyan]🔍 Searching for:[/bold cyan] [yellow]{query}[/yellow]\n")

    try:
        resp = httpx.get(
            f"{API_BASE}/search",
            params={"q": query},
            timeout=10,
        )
        resp.raise_for_status()
        results = resp.json()

        if not results:
            console.print("[dim]No results found.[/dim]")
            return

        table = Table(title=f"Search Results ({len(results)} found)", border_style="cyan")
        table.add_column("#", style="dim", width=4)
        table.add_column("Relevant URL", style="green", max_width=60)
        table.add_column("Origin URL", style="blue", max_width=40)
        table.add_column("Depth", justify="center", style="yellow")
        table.add_column("Score", justify="right", style="magenta")

        for i, r in enumerate(results, 1):
            table.add_row(
                str(i),
                r.get("relevant_url", ""),
                r.get("origin_url", ""),
                str(r.get("depth", "")),
                str(r.get("score", "")),
            )

        console.print(table)
    except httpx.ConnectError:
        console.print("[bold red]✗ Cannot connect to server. Is it running on :8000?[/bold red]")
    except Exception as e:
        console.print(f"[bold red]✗ Error: {e}[/bold red]")


# ── status ──────────────────────────────────────────────────────────────────

def cmd_status() -> None:
    """Poll GET /status every 2 seconds and display a live dashboard."""
    console.print("[bold cyan]📊 Live Status Dashboard[/bold cyan] (Ctrl+C to stop)\n")

    def build_dashboard() -> Panel:
        try:
            resp = httpx.get(f"{API_BASE}/status", timeout=5)
            resp.raise_for_status()
            data = resp.json()
        except Exception:
            return Panel("[bold red]Cannot connect to server[/bold red]", title="Status")

        bp_status = "[bold red]⚠ ACTIVE[/bold red]" if data.get("back_pressure_active") else "[bold green]○ Inactive[/bold green]"

        lines = [
            f"[bold]Pages Indexed:[/bold]      {data.get('pages_indexed', 0)}",
            f"[bold]Queue Depth:[/bold]        {data.get('queue_depth', 0)}",
            f"[bold]Back-pressure:[/bold]      {bp_status}",
            f"[bold]Crawl Rate:[/bold]         {data.get('crawl_rate_per_sec', 0):.2f} pages/sec",
            f"[bold]Active Sessions:[/bold]    {len(data.get('active_sessions', []))}",
        ]

        sessions = data.get("active_sessions", [])
        if sessions:
            lines.append("")
            lines.append("[bold underline]Active Sessions:[/bold underline]")
            for s in sessions:
                lines.append(f"  • ID {s['id']}: {s['origin_url']} (depth={s['max_depth']}, status={s['status']})")

        content = "\n".join(lines)
        return Panel(content, title="🕷️  Crawler Status", border_style="cyan", padding=(1, 2))

    try:
        with Live(build_dashboard(), console=console, refresh_per_second=0.5) as live:
            while True:
                time.sleep(2)
                live.update(build_dashboard())
    except KeyboardInterrupt:
        console.print("\n[dim]Dashboard stopped.[/dim]")


# ── main ────────────────────────────────────────────────────────────────────

def main() -> None:
    if len(sys.argv) < 2:
        console.print(
            Panel(
                "[bold]Usage:[/bold]\n"
                "  python cli.py [cyan]index[/cyan]  <url> <depth>   — Start a crawl\n"
                "  python cli.py [cyan]search[/cyan] <query>         — Search indexed pages\n"
                "  python cli.py [cyan]status[/cyan]                 — Live status dashboard",
                title="Multi-Agent Crawler CLI",
                border_style="blue",
            )
        )
        sys.exit(1)

    command = sys.argv[1].lower()

    if command == "index":
        if len(sys.argv) < 4:
            console.print("[red]Usage: python cli.py index <url> <depth>[/red]")
            sys.exit(1)
        cmd_index(sys.argv[2], int(sys.argv[3]))

    elif command == "search":
        if len(sys.argv) < 3:
            console.print("[red]Usage: python cli.py search <query>[/red]")
            sys.exit(1)
        query = " ".join(sys.argv[2:])
        cmd_search(query)

    elif command == "status":
        cmd_status()

    else:
        console.print(f"[red]Unknown command: {command}[/red]")
        sys.exit(1)


if __name__ == "__main__":
    main()
