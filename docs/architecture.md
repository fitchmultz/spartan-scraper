# Architecture

## First principles

- Fetch is separate from extraction.
- Crawl is a controlled graph walk, not a firehose.
- The same job model powers CLI, TUI, and API.
- State is local and persistent; results are immutable artifacts.
- Ignore robots.txt by default; throttling is still available for stability.

## Core modules

- `fetch`: HTTP fetcher + headless Chromium/Playwright fetchers.
- `extract`: HTML → text/metadata/links.
- `crawl`: BFS crawler with host scoping and depth/limit controls.
- `jobs`: persistent store + queue + runner.
- `api`: REST API aligned to `api/openapi.yaml`.
- `ui/tui`: job list + status dashboard.
- `web`: Web UI consuming generated API client.
- `research`: multi-source workflow (scrape/crawl → evidence → summary).
- `mcp`: stdio server exposing tools for agent orchestration.
- Auth profiles live in `DATA_DIR/profiles.json` and can be referenced by CLI.
- Exporter can emit markdown or csv from stored job artifacts.
- Scheduler runs interval-based jobs and persists schedules in `DATA_DIR/schedules.json`.

## Adaptive Render Pipeline

The fetcher uses an adaptive strategy to optimize for performance and reliability:

1. **Profile Check**: Checks `DATA_DIR/render_profiles.json` for per-host rules (forced engine, timeouts, blocking).
2. **HTTP Probe**: By default, attempts a fast HTTP GET.
3. **Detection**: Analyzes the HTML for "JS-heavy" signals (SPA roots, noscript warnings, high script/text ratio).
4. **Escalation**: If the page is detected as dynamic (or returns 403/401 bots blocks), it escalates to a headless browser (Chromedp or Playwright).
5. **Optimization**:
   - Blocks wasteful resources (images, fonts, media, stylesheets) by default or policy.
   - Uses adaptive wait strategies (DOM ready, network idle, selector visible, content stability).

## Execution flow

1. CLI/API create a job and persist it.
2. Job runner fetches HTML via HTTP or headless.
3. Extractor emits text/metadata/links.
4. Results are written to JSONL under `DATA_DIR/jobs/<id>`.
5. Status updates are persisted and served to UI/TUI.

## Auth model

- Header + cookie injection at the HTTP layer.
- Basic auth for direct endpoints.
- Form login via headless Chromium or Playwright (selectors provided).

## Interfaces

- CLI: `spartan scrape|crawl|server|tui`.
- API: `/v1/scrape`, `/v1/crawl`, `/v1/jobs`.
- UI: Web app for job submission + status.
