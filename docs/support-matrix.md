# Support Matrix

## Supported 1.0 Core

- Single-node local deployment
- SQLite plus local artifact storage
- REST API, WebSocket events, and MCP
- Scrape, crawl, and research jobs
- Templates, pipeline JS, auth vault, schedules, batches, chains, watches, retention, and backup/restore
- Exports: `json`, `jsonl`, `csv`, `md`, `xlsx`
- Direct saved-result exports use one shared `format` / `shape` / `transform` contract across REST, Web UI, CLI, and MCP; `shape` and `transform` are mutually exclusive.

## AI Interface Policy

- REST is the canonical machine surface for bounded AI authoring and job-integrated AI extraction.
- Web UI is the primary interactive AI surface.
- CLI and MCP expose both job-launching AI controls and dedicated AI preview, template generation, template debugging, render-profile generation, render-profile debugging, pipeline-JS generation, pipeline-JS debugging, research-refinement, export-shaping, saved-result transform-generation, direct saved-result export contracts, and recurring export-contract workflows.
- Live and batch scrape/crawl/research submission now route operator-facing request-to-spec conversion through `internal/submission` across REST, direct CLI batch execution, Web UI, MCP, schedules, chains, and watches for shared auth, webhook, proxy transport, device emulation, screenshot capture, network interception, and pipeline defaults.
- Optional proxy-pool execution is a supported runtime path with shared status inspection plus per-request direct-proxy or proxy-hint selection controls across REST, Web UI, CLI, and MCP.
- Proxy pooling is fully opt-in: leaving `PROXY_POOL_FILE` unset keeps it disabled and noise-free, while explicit non-empty `PROXY_POOL_FILE` values still surface configuration mistakes instead of silently pretending proxy-backed execution is active.
- Request-scoped multimodal image context for AI authoring is supported on REST, Web, CLI, and MCP: operators can attach bounded uploaded/pasted images directly, and URL-based flows can additionally capture screenshots when an image-capable pi route is available.
- TUI is an intentionally limited operations and inspection surface. It may display AI metadata already persisted on jobs, but it does not carry dedicated AI preview, AI template generation, AI template debugging, AI render-profile generation, AI render-profile debugging, AI pipeline-JS generation, AI pipeline-JS debugging, research-refinement, export-shaping, saved-result transform-generation, or agent-session workflows.
- Feature-parity work should target REST, Web UI, CLI, and MCP first; the TUI is not a standing parity target unless the roadmap explicitly elevates it.

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
