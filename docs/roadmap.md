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

1. Remove the hidden AI rail footprint from `/jobs/new`
   - Collapse the hidden assistant affordance so it does not reserve a dedicated sidebar column on short laptop viewports.
   - Keep presets and sticky actions visible without introducing a new responsive mode.

2. Normalize the remaining template-authoring error copy
   - Replace the raw `String(response.error)` fallbacks in `useTemplateBuilder` and `VisualSelectorBuilder` with `getApiErrorMessage` copy.
   - Keep save, fetch, and selector-test failures operator-readable across the template flow.

3. Isolate dedup panel state by active tab
   - Scope dedup errors, stale results, and refresh failures to the active panel in `DedupExplorer`.
   - Prevent a failed search, history lookup, or stats refresh from polluting the other dedup surfaces.

4. Backfill focused regressions for authoring and dedup failures
   - Add only the direct coverage needed for template save, visual-selector fetch/test, and dedup search/history/stats failures.
   - Assert operator-visible recovery copy, panel-local failure handling, and draft/result preservation instead of implementation details.

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
