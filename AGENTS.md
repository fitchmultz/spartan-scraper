# AGENTS.md

- Primary language: Go (CLI + API + TUI). Frontend: TypeScript (Vite + React).
- Local gate: `make ci` (runs generate, format, type-check, lint, build, test).
- API contract: `api/openapi.yaml` → generate TS client with `make generate` (hey-api openapi-ts).
- Data storage: local on-disk job store under `DATA_DIR` (default `.data`).
- Ignore robots.txt by design (do not add compliance logic without explicit request).
