# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.

## Now

- Add agentic `research` powered by `pi`.
- Keep deterministic `internal/research` as the baseline path; current AI research remains additive through the existing extraction/evidence workflow unless a future roadmap item explicitly restructures it.
- Reuse Spartan's existing evidence collection and fetch/extract primitives instead of bypassing them with a free-form agent loop.

## Next

- Revisit interface symmetry for any new AI capability as part of feature design, rather than shipping API-only or Web-only by default.

## Later

- Revisit multimodal/template-debug loops once image-capable routes prove stable in production-like usage.
- Broaden `pi` usage beyond extraction/template generation where an agent harness improves real workflows.
