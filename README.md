# Spartan Scraper

A high‑performance, Go‑first web scraping + automation standard for all future projects.

## Goals

- Single pages, full websites, and deep research workflows.
- Works for static HTML and JS‑heavy sites (headless Chromium or Playwright).
- Unified interfaces: CLI, TUI, and Web UI.
- Clean API contract (OpenAPI) with generated TS client.
- Local, self‑hosted, no SaaS dependencies.

## Quickstart

```bash
make install
make generate
make build
./bin/spartan --help
```

### CLI examples

```bash
# Single page scrape (HTTP)
./bin/spartan scrape \
  --url https://example.com \
  --out ./out/example.json

# Headless scrape with login (form-based)
./bin/spartan scrape \
  --url https://example.com/dashboard \
  --headless \
  --playwright \
  --login-url https://example.com/login \
  --login-user-selector '#email' \
  --login-pass-selector '#password' \
  --login-submit-selector 'button[type=submit]' \
  --login-user you@example.com \
  --login-pass '***' \
  --out ./out/dashboard.json

# Crawl a site (depth-limited)
./bin/spartan crawl \
  --url https://example.com \
  --max-depth 2 \
  --max-pages 200 \
  --out ./out/site.jsonl

# Deep research
./bin/spartan research \
  --query "pricing model" \
  --urls https://example.com,https://example.com/docs \
  --out ./out/research.jsonl

# MCP server (stdio)
./bin/spartan mcp

# Auth profiles
./bin/spartan auth set --name acme --auth-basic user:pass --header "X-API: token"
./bin/spartan scrape --url https://example.com --auth-profile acme

# Export results
./bin/spartan export --job-id <id> --format md --out ./out/report.md

# Schedules
./bin/spartan schedule add --kind scrape --interval 3600 --url https://example.com
./bin/spartan schedule list

# Run API server + background worker
./bin/spartan server

# Launch TUI
./bin/spartan tui
```

### Web UI

```bash
make web-dev
```

Open `http://localhost:5173` for the UI (the API is proxied to `PORT` from `.env`).

## Architecture at a glance

- `cmd/spartan`: main entrypoint for CLI/TUI/API server.
- `internal/fetch`: HTTP + headless fetchers (Chromedp + Playwright).
- `internal/extract`: HTML parsing + text/metadata extraction.
- `internal/crawl`: BFS crawler with depth/limit and domain scoping (robots ignored).
- `internal/jobs`: persistent job store + queue runner.
- `internal/api`: HTTP API + OpenAPI contract.
- `web`: Vite + React UI; generated API client under `web/src/api`.
- `internal/research`: multi-source workflow (scrape/crawl → evidence → summary).
- `internal/mcp`: MCP stdio server for agent orchestration.
- `internal/exporter`: exports results to markdown/csv/json.
- `internal/scheduler`: recurring job runner with interval schedules.

## Notes

- Robots.txt is intentionally ignored by design.
- Auth support: headers, cookies, basic auth, and form login (headless).
- Auth profiles are stored in `.data/profiles.json`.
- Rate limiting + retries are configurable via `.env`.
- Playwright can be enabled with `USE_PLAYWRIGHT=1` or `--playwright` (CLI/API). Install browsers with:

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install --with-deps
```

## Toolchain

Pinned in `.tool-versions`:

- Go `1.25.6`
- Node `24.13.0`
- pnpm `10.28.0`

## Local CI

```bash
make ci
```
