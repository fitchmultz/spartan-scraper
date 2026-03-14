# Support Matrix

## Supported 1.0 Core

- Single-node local deployment
- SQLite plus local artifact storage
- REST API, WebSocket events, and MCP
- Scrape, crawl, and research jobs
- Templates, pipeline JS, auth vault, schedules, batches, chains, watches, retention, and backup/restore
- Exports: `json`, `jsonl`, `csv`, `md`, `xlsx`

## AI Interface Policy

- REST is the canonical machine surface for bounded AI authoring and job-integrated AI extraction.
- Web UI is the primary interactive AI surface.
- CLI and MCP expose both job-launching AI controls and dedicated AI preview, template generation, template debugging, render-profile generation, render-profile debugging, pipeline-JS generation, and pipeline-JS debugging workflows.
- Multimodal screenshot context for AI authoring is supported on REST, Web, CLI, and MCP when the workflow fetches a URL through a headless browser and an image-capable pi route is available.
- TUI is an operations and inspection surface. It may display AI metadata already persisted on jobs, but it does not carry dedicated AI preview, AI template generation, AI template debugging, AI render-profile generation, AI render-profile debugging, AI pipeline-JS generation, AI pipeline-JS debugging, or agent-session workflows.

## Release Guarantee

- The supported 1.0 core is the only compatibility target for release candidates and patch fixes.
- Removed surfaces are not part of the release contract and may be absent from storage, API payloads, docs, and generated clients.

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
