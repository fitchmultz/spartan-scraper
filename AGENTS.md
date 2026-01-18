# AGENTS.md

- Primary language: Go (CLI + API + TUI). Frontend: TypeScript (Vite + React).
- Local gate: `make ci` (runs generate, format, type-check, lint, build, test).
- API contract: `api/openapi.yaml` → generate TS client with `make generate` (hey-api openapi-ts).
- Data storage: local on-disk job store under `DATA_DIR` (default `.data`).
- Ignore robots.txt by design (do not add compliance logic without explicit request).
- Playwright is optional for JS-heavy pages (`USE_PLAYWRIGHT=1` or `--playwright`).
- Extraction pipeline is centralized in `internal/extract`. Templates live in `DATA_DIR/extract_templates.json`.
- Pipeline hooks and plugin contracts live in `internal/pipeline` (pre/post fetch/extract/output + transformers).
- Headless per-target JS is configured in `DATA_DIR/pipeline_js.json`.
