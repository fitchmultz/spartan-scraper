# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.

## Now

- Align AI extraction controls for scrape/crawl job submission across API, Web UI, CLI, and MCP where those job-launching interfaces already exist.

## Next

- Add AI capabilities to additional product features where they materially improve outcomes and fit the interface, starting with research.
- Keep this additive first: deterministic `internal/research` remains the baseline path unless a future roadmap item explicitly replaces or restructures it.
- Reuse Spartan's existing evidence collection and fetch/extract primitives instead of bypassing them with a free-form agent loop.
- Add agentic `research` powered by `pi`.

## Later

- Revisit interface symmetry for any new AI capability as part of feature design, rather than shipping API-only or Web-only by default.
- Revisit multimodal/template-debug loops once image-capable routes prove stable in production-like usage.
- Broaden `pi` usage beyond extraction/template generation where an agent harness improves real workflows.
