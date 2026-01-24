# AGENTS.md

- Primary language: Go (CLI + API + TUI). Frontend: TypeScript (Vite + React).
- Local gate: `make ci` (runs generate, format, type-check, lint, build, test).
  - **CRITICAL**: NEVER end a turn with a failing `make ci`. If `make ci` fails, fix the failures before completing your work.
  - Flaky e2e tests may be retried up to 3 times before considering them a real failure.
- API contract: `api/openapi.yaml` → generate TS client with `make generate` (hey-api openapi-ts).
- Data storage: local on-disk job store under `DATA_DIR` (default `.data`).
- Ignore robots.txt by design (do not add compliance logic without explicit request).
- Playwright is optional for JS-heavy pages (`USE_PLAYWRIGHT=1` or `--playwright`).
- Extraction pipeline is centralized in `internal/extract`. Templates live in `DATA_DIR/extract_templates.json`.
- Pipeline hooks and plugin contracts live in `internal/pipeline` (pre/post fetch/extract/output + transformers).
- Headless per-target JS is configured in `DATA_DIR/pipeline_js.json`.

## Package Structure
- `cmd/spartan`: Main entry point for the CLI.
- `internal/api`: REST API server and route handlers.
- `internal/auth`: Auth profile management and vault.
- `internal/cli`: CLI subcommand implementations.
- `internal/config`: Global configuration and logging.
- `internal/crawl`: Concurrent website crawling logic.
- `internal/extract`: HTML content extraction and normalization.
- `internal/fetch`: Content fetching (HTTP, Chromedp, Playwright).
- `internal/jobs`: Job manager and worker pool.
- `internal/model`: Shared domain models and constants.
- `internal/pipeline`: Pipeline hooks, processors, and transformers.
- `internal/store`: Persistent storage for jobs and crawl states.
- `internal/ui/tui`: Terminal User Interface.
