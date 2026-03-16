# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Collapse the remaining CLI batch preflight validation duplication onto `internal/submission` so API-backed and direct batch submissions surface the same request-count, URL, and shared-option errors before execution.
- Remove the temporary transitive Go dependency overrides in `go.mod` as soon as upstream parent modules absorb those newer tags; keep `make audit-deps` green so override cleanup happens immediately once the parent graph catches up.
