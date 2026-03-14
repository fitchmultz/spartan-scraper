# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.

## Now

- Extend the shared `/v1/ai/*` authoring surface into bounded research-output refinement workflows that help operators rewrite, condense, or normalize collected evidence across API, Web, CLI, and MCP without introducing free-form agent loops.

## Next

- Add bounded export-shaping authoring flows so operators can generate or tune export-ready field sets, summaries, and formatting hints before running recurring exports.

## Later

- Revisit direct image upload/paste support for AI authoring once request-envelope constraints and artifact retention expectations are settled.
