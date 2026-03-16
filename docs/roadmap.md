# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Centralize create-time webhook URL validation across watch, job, and export-schedule entry points so operators get the same early syntax errors everywhere, while runtime delivery remains the enforcement boundary for SSRF checks and delivery-time IP pinning.
- Remove the temporary transitive Go dependency overrides in `go.mod` as soon as upstream parent modules absorb those newer tags; keep `make audit-deps` green so override cleanup happens immediately once the parent graph catches up.
