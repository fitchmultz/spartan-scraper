# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Add system-level parity coverage that submits equivalent scrape/crawl/research requests through REST, CLI, and MCP, then diffs the persisted typed specs so request-contract drift is caught before release.
- Unify chain node submission and watch-triggered job creation on the same operator-facing execution model used by live jobs and schedules, eliminating the remaining contract-special cases before adding more automation surfaces.
- Normalize job and batch response shaping across REST, Web UI, and MCP so downstream clients can consume one stable status/result envelope without transport-specific branching.
