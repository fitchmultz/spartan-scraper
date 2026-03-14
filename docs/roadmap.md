# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.

## Now

- Add bounded debug/tuning loops for AI-authored runtime automation configs (render profiles and pipeline JS) using live page rechecks, deterministic validation, and shared retry patterns across API, Web, CLI, and MCP.

## Next

- Extend the shared `/v1/ai/*` authoring surface into other bounded operator workflows where `pi` materially improves outcomes, such as export shaping or research-output refinement, without introducing free-form agent loops.

## Later

- Revisit direct image upload/paste support for AI authoring once request-envelope constraints and artifact retention expectations are settled.
