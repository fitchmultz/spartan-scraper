# Support Matrix

## Supported 1.0 Core

- Single-node local deployment
- SQLite plus local artifact storage
- REST API, WebSocket events, and MCP
- Scrape, crawl, and research jobs
- Templates, pipeline JS, auth vault, schedules, batches, chains, watches, retention, and backup/restore
- Exports: `json`, `jsonl`, `csv`, `md`, `xlsx`

## Removed

- GraphQL
- WASM plugins
- Distributed Redis mode
- Multi-user/workspaces
- Browser extension
- Replay tooling
- Template A/B testing and template analytics dashboards
- Feed monitoring
- Cloud/database exporters
- HAR, PDF, and Parquet exporters

## Storage Policy

- `balanced-1.0-2026-03-11` is a hard cutover schema marker.
- Pre-cutover databases are rejected on startup.
- Reset or migrate old data directories outside this repo before running the 1.0 build.
