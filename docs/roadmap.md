# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Vet the optional proxy-pool path end to end, confirm whether the current configuration and runtime behavior still work, and add durable verification coverage so future changes cannot silently break proxy-backed execution.

## Next

- Revisit direct image upload/paste support for AI authoring once request-envelope constraints and artifact retention expectations are settled.

## Later

- Extend bounded AI authoring to more saved result-artifact workflows only after proxy-pool vetting lands with explicit runtime/documentation guarantees and export-shaping remains stable across API, Web, CLI, and MCP.
