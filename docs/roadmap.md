# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Add persistent transform configuration to export schedules across API, Web UI, CLI, and MCP so saved transformed exports and recurring exports share one deterministic contract.

## Next

- Group the next saved-result authoring contract changes around result-derived export workflows that can reuse the new transform helpers without reintroducing ad hoc interface drift.

## Later

- Deepen operator-facing proxy selection controls only if a concrete runtime need remains after the current global proxy-pool contract and verification coverage prove sufficient in day-to-day use.
