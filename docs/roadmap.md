# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Close the highest-value remaining operator-surface gaps across API, Web UI, CLI, MCP, and TUI for batches, watches, export schedules, and webhook inspection.
- Expand operator observability so run history, export outcomes, and webhook delivery failures are actionable without digging through host-local files or internal state.
