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

1. Normalize template-authoring failure copy
   - Replace the remaining raw `String(response.error)` fallbacks in `useTemplateBuilder` and `VisualSelectorBuilder` with `getApiErrorMessage`.
   - Keep save, fetch, and selector-test failures operator-readable and action-oriented.

2. Scope dedup state by active tab
   - Keep `DedupExplorer` search, history, and stats errors isolated to the active panel.
   - Prevent stale results or refresh failures in one tab from mutating the others.

3. Backfill focused authoring and dedup regressions
   - Add only the direct tests needed for template save, visual-selector fetch/test, and dedup search/history/stats failures.
   - Assert operator-visible recovery and panel-local state preservation instead of implementation details.

4. Unify long-running validation helpers
   - Extract one shared helper for spawning and tearing down repo-owned test processes across `internal/e2e`, `internal/system`, and heavy validation paths.
   - Keep API envelope parsing aligned between PR-safe and heavy validation coverage.

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
