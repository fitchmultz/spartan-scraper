# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Extend bounded AI authoring to more saved result-artifact workflows now that direct image support, export shaping, and proxy-pool runtime guarantees are stable across API, Web, CLI, and MCP.

## Next

- Review which persisted-result workflows should gain the next bounded multimodal authoring passes so future contract changes can be grouped and regenerated together.

## Later

- Deepen operator-facing proxy selection controls only if a concrete runtime need remains after the current global proxy-pool contract and verification coverage prove sufficient in day-to-day use.
