# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Treat the Web UI as a first-class operator surface. When parity work and workflow clarity compete, prioritize the product workflow that helps operators complete real tasks faster and with less confusion.
- Prefer feature symmetry across the primary product interfaces that carry the main operator and automation workflows: API, Web UI, CLI, and MCP, but do not preserve a poor Web UI solely for parity.
- Treat the TUI as an intentionally limited local inspection surface, not a feature-parity target, unless this roadmap explicitly says otherwise.
- Add AI enablement where it improves a real scraping, template-authoring, or results-analysis workflow; do not force AI into surfaces where it adds little operational value.
- Preserve route-per-major-feature information architecture at the top level, then use sub-routing or explicit in-route navigation inside complex Web surfaces when that materially improves clarity.
- Treat interface asymmetry as intentional only when this roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.
- Put meaningful operator-facing product work ahead of maintenance, cleanup, and policy reminders.
- Treat focused failure-path dogfooding as acceptance criteria for major operator workflow cutovers, not as a standalone roadmap epic.

## Next

1. Consolidate browser-runtime request builders in the Web client
   - Remove the repeated headless/playwright/timeout/extract-merging request shaping from `web/src/lib/form-utils.ts`, `web/src/lib/batch-utils.ts`, and the authoring tools.
   - Keep one shared browser-runtime serializer for single-job, batch, and authoring requests before more UI refactors.

2. Reuse shared browser-runtime controls across authoring tools
   - Replace bespoke headless/playwright state and toggles in template preview, template assistant, visual selector, render-profile AI, pipeline-JS AI, and job-submission assistant with shared state and `BrowserExecutionControls`.
   - Keep one dependency rule for headless-gated capabilities such as Playwright, screenshots, device emulation, and network interception.

3. Decide whether dedup stays maintenance-only or gets an explicit crawl indexing path
   - If operators need live dedup data, add one deliberate crawl/indexing contract across API, CLI, Web, and persisted job specs instead of reviving stray flags.
   - Otherwise keep dedup as a maintenance API and delete any remaining crawl-only cross-job duplicate plumbing.

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
