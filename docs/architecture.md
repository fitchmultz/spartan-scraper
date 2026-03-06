# Architecture

## Quick overview (2-minute read)

Spartan Scraper is a **local-first scraping platform** built around a single job model shared by CLI, API, TUI, Web UI, and MCP.

- Interfaces submit jobs into a persistent local store.
- A job runner executes fetch + extract + pipeline stages with concurrency/rate controls.
- Results are written as immutable artifacts under `DATA_DIR/jobs/<id>`.
- Status/events are exposed to API, UI, and TUI consumers.

### Control/data flow

1. **Submit**: CLI/API/UI creates a typed job (`scrape`, `crawl`, `research`, etc.).
2. **Persist + queue**: job metadata is stored; workers pick pending jobs.
3. **Fetch**: HTTP first, escalate to headless engines when content is JS-heavy.
4. **Extract + transform**: extraction templates, pipeline hooks, and optional plugins/scripts enrich output.
5. **Store + serve**: artifacts and status are persisted and made available through REST/WS/GraphQL and UI surfaces.

### Key decisions and trade-offs

- **Local persistence over managed services**: simpler operations and privacy-friendly defaults, at the cost of single-node scaling limits.
- **HTTP-first adaptive render pipeline**: minimizes resource usage while preserving JS-heavy site reliability.
- **Opt-in heavy features** (Playwright, e2e stress checks): keeps default developer workflow fast/deterministic.
- **Contract-driven API client generation**: tighter backend/frontend compatibility with explicit generate step in CI.

## First principles

- Fetch is separate from extraction.
- Crawl is a controlled graph walk, not a firehose.
- The same job model powers CLI, TUI, and API.
- State is local and persistent; results are immutable artifacts.
- Ignore robots.txt by default (opt-in compliance is available); throttling is still available for stability.

## System diagram (high-level)

```text
User Interfaces
  ├─ CLI (spartan)
  ├─ TUI
  ├─ Web UI (React)
  └─ API clients (REST / WS / GraphQL)
            │
            v
Application Core (internal/*)
  ├─ fetch (http/chromedp/playwright)
  ├─ extract
  ├─ crawl
  ├─ jobs / queue / scheduler
  ├─ research / exporter / pipeline / plugins
  └─ api / mcp / auth
            │
            v
Persistence (DATA_DIR)
  ├─ jobs artifacts
  ├─ auth vault
  ├─ templates
  ├─ schedules
  └─ plugin state
```

## Core modules

- `fetch`: HTTP fetcher + headless Chromium/Playwright fetchers.
- `extract`: HTML → text/metadata/links.
- `crawl`: BFS crawler with host scoping and depth/limit controls.
- `jobs`: persistent store + queue + runner.
- `api`: REST API aligned to `api/openapi.yaml`.
- `ui/tui`: job list + status dashboard.
- `web`: Web UI consuming generated API client.
- `research`: multi-source workflow (scrape/crawl → evidence → summary → simhash dedup → clustering → citations + confidence).
- `mcp`: stdio server exposing tools for agent orchestration.
- Auth vault lives in `DATA_DIR/auth_vault.json` (profiles, inheritance, presets).
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

## Pipeline hooks + plugins

- `internal/pipeline` defines the standardized plugin interface and hook registry.
- Hooks are executed at pre/post fetch, pre/post extract, and pre/post output.
- Output transformers run after pre-output hooks and before post-output hooks.
- JS per-target scripts are loaded from `DATA_DIR/pipeline_js.json` and applied during headless fetch.

## Plugin System (WASM)

The plugin system enables third-party extensions via sandboxed WASM plugins:

- **Package**: `internal/plugins`
- **Storage**: `DATA_DIR/plugins/<name>/` (manifest.json + plugin.wasm)
- **Runtime**: wazero (pure Go, no CGO)
- **Security**: WASI preview1 with explicit permission model

### Plugin lifecycle

1. **Discovery**: Loader scans `DATA_DIR/plugins/` for manifest.json files
2. **Validation**: Manifest validated (name, version, hooks, permissions, wasm_path)
3. **Loading**: WASM module compiled and cached
4. **Instantiation**: New instance per hook execution with isolated memory
5. **Execution**: JSON-serialized input/output via exported functions

### Hook interface

Plugins export functions matching hook names:
- `pre_fetch(input_ptr, input_len) -> output_ptr_with_len`
- `post_fetch(input_ptr, input_len) -> output_ptr_with_len`
- `pre_extract`, `post_extract`, `pre_output`, `post_output`

Input/output is JSON-encoded. Memory management via exported `malloc`/`free`.

### Host functions

Available to plugins based on granted permissions:
- `log(msg_ptr, msg_len)` - Always available
- `get_config(key_ptr, key_len) -> value_ptr_with_len` - Always available
- `http_request(...)` - Requires `network` permission
- `file_access(...)` - Requires `filesystem` permission (restricted to plugin directory)
- `get_env(key_ptr, key_len)` - Requires `env` permission (only `SPARTAN_PLUGIN_*` vars)

### Manifest schema

```json
{
  "name": "header-injector",
  "version": "1.0.0",
  "description": "Injects custom headers",
  "author": "Author Name",
  "hooks": ["pre_fetch"],
  "permissions": ["network"],
  "wasm_path": "plugin.wasm",
  "config": { "headers": {} },
  "enabled": true,
  "priority": 10
}
```

### JS registry example

```json
{
  "scripts": [
    {
      "name": "app-login",
      "hostPatterns": ["app.example.com"],
      "engine": "playwright",
      "preNav": "localStorage.setItem('consent','true')",
      "postNav": "window.scrollTo(0, document.body.scrollHeight)",
      "selectors": ["#app", "main"]
    }
  ]
}
```

## Execution flow

1. CLI/API create a job and persist it.
2. Job runner fetches HTML via HTTP or headless.
3. Extractor emits text/metadata/links.
4. Results are written to JSONL under `DATA_DIR/jobs/<id>`.
5. Status updates are persisted and served to UI/TUI.

## Auth model

- Unified auth vault with profile inheritance and per-target presets.
- Headers, cookies, basic auth, bearer/api_key tokens, and headless login flows.
- Env overrides supported via `AUTH_*` variables and applied during resolution.
- Query tokens are appended to request URLs where supported.

## Interfaces

- CLI: `spartan scrape|crawl|research|server|tui|watch|batch|chains|feed|retention|export-schedule`.
- API:
  - REST: `/v1/*` (OpenAPI-backed, see `api/openapi.yaml`)
  - WebSocket: `/v1/ws` (real-time job events + metrics)
  - GraphQL: `/graphql` + `/graphql/playground` (not in OpenAPI; see schema in `internal/api/graphql/schema.go`)
- UI: Web app for job submission + status.
