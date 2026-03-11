# Architecture

Spartan Scraper is a single-node, local-first scraping workbench. The stable 1.0 boundary is intentionally narrow:

- scrape, crawl, and research jobs
- auth vault and OAuth helpers
- templates and pipeline JavaScript
- schedules, batches, chains, watches, and webhooks
- REST, WebSocket, Web UI, CLI, TUI, and MCP
- local artifacts, retention, backup, and restore
- exports in `json`, `jsonl`, `csv`, `md`, and `xlsx`

Removed from the supported product surface: GraphQL, plugins, distributed Redis mode, multi-user/workspaces, browser extension, feed monitoring, replay tooling, template A/B metrics, and cloud/database exporters.

## System model

1. An interface submits a job: CLI, Web UI, REST, scheduler, or MCP.
2. The job manager persists the job to the local SQLite store.
3. Workers execute fetch, extract, pipeline, and optional headless/browser steps.
4. Results are written under `DATA_DIR/jobs/<job-id>/`.
5. REST, WebSocket, TUI, and MCP read the same persisted state and artifacts.

Everything runs against one local data directory. There is no supported distributed mode.

## Supported interfaces

- REST: canonical machine API defined by [`api/openapi.yaml`](../api/openapi.yaml)
- WebSocket: live job and manager events on `/v1/ws`
- MCP: stdio server for agent orchestration
- Web UI: route-based shell for jobs, templates, automation, and settings
- CLI/TUI: local operator and scripting interfaces

GraphQL is not part of the architecture anymore.

## Core packages

- `cmd/spartan`
  - Process entrypoint for CLI, server, TUI, and MCP.
- `internal/api`
  - REST handlers, WebSocket endpoint, and request/response shaping.
- `internal/jobs`
  - Job creation, queueing, execution, cancellation, recovery, and event publication.
- `internal/fetch`
  - HTTP-first fetching with Chromedp/Playwright escalation, auth, rate limiting, and browser helpers.
- `internal/extract`
  - HTML normalization, extraction templates, and AI-assisted extraction helpers.
- `internal/crawl`
  - Same-host bounded crawling with dedup and optional robots handling.
- `internal/research`
  - Derived multi-source workflow built on scrape/crawl primitives.
- `internal/scheduler`
  - Interval schedules and export schedules.
- `internal/exporter`
  - Artifact export for `json`, `jsonl`, `csv`, `md`, and `xlsx`.
- `internal/auth`
  - Auth profiles, presets, login flow definitions, OAuth helpers, and API key support.
- `internal/store`
  - SQLite-backed persistence for jobs, crawl state, automation records, and analytics still retained in the core product.
- `internal/mcp`
  - MCP tool definitions and handlers.
- `web`
  - Vite + React UI using the generated OpenAPI client.

## Persistence and artifacts

The canonical runtime model is:

- SQLite database: `DATA_DIR/jobs.db`
- Job artifacts: `DATA_DIR/jobs/<job-id>/`
- Auth vault: `DATA_DIR/auth_vault.json`
- Render profiles: `DATA_DIR/render_profiles.json`
- Templates: `DATA_DIR/extract_templates.json`
- Pipeline JS: `DATA_DIR/pipeline_js.json`
- Schedules/export schedules: files and SQLite-backed state under `DATA_DIR`

Artifacts are designed to be inspectable on disk. Result files remain the source for exports and downstream tooling.

## Storage compatibility policy

Balanced 1.0 is a hard cutover.

- A fresh data directory is initialized with the current storage schema marker.
- An existing `jobs.db` without the Balanced 1.0 schema marker is treated as legacy and rejected at startup.
- The supported migration path is: back up the old data directory, then reset to a fresh Balanced 1.0 data directory.

This avoids silently opening pre-cutover state under a materially different product boundary.

## Execution path

### Scrape

- Validate URL and execution settings.
- Fetch with HTTP first.
- Escalate to headless Chromium or Playwright when the target requires it.
- Apply extraction and pipeline processing.
- Persist one or more result records to the job artifact file.

### Crawl

- Validate URL, bounds, and crawl options.
- Walk same-host pages with depth/page caps.
- Reuse the same fetch/extract pipeline as scrape.
- Persist one result record per crawled page.

### Research

- Validate query and source URLs.
- Gather evidence from scrape/crawl-style fetches.
- Cluster, summarize, and cite results.
- Persist derived research output as job artifacts.

## Web UI information architecture

The web shell is route-based and scoped to the retained product:

- `/jobs`
- `/jobs/new`
- `/jobs/:id`
- `/templates`
- `/automation`
- `/settings`

Deleted product areas are not hidden behind feature flags; they are absent from the navigation and render tree.

## Operational model

The official deployment shape is:

- single process or single host
- local disk
- SQLite
- trusted operator
- optional API key protection for non-loopback binds

There is no supported multi-user authorization boundary or cluster story in this cutover.

## CI and validation

The required local gate is `make ci`.

That gate covers:

- OpenAPI generation
- Go and web formatting/lint/type-check
- Go build
- Go tests excluding heavy E2E
- web Vitest suite

Heavier browser and stress validation remains in `make ci-slow`.
