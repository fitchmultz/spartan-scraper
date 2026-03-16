# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Keep the remaining managed transitive Go overrides synchronized between `go.mod` and `scripts/go_transitive_override_audit.mjs`, and remove entries immediately as upstream parent modules absorb those newer tags.
