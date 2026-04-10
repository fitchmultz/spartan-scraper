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

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.

## Audit-Driven Priorities (2026-04-01)

See [docs/codebase-audit-2026-04-01.md](codebase-audit-2026-04-01.md) for the full audit catalog, metrics, and evidence.

1. **Harden lifecycle cleanup and long-running backend control loops.**
   - Make Playwright availability checks cancellable and cleanup-safe.
   - Stop deleting job rows before artifact cleanup succeeds.
   - Persist analytics snapshots outside collector critical sections.
   - Ship these together with deterministic tests that exercise timeout, cleanup, and storage-failure paths.
   - File split prerequisite complete: `webhook/dispatcher`, `api/diagnostic_status`, and `watch/watch` now have focused sibling files so lifecycle hardening changes have smaller blast radius.

2. **Split the web job-authoring state model into smaller, feature-owned slices.**
   - Break `useFormState` into narrower runtime/auth/AI/intercept modules instead of one shared god hook.
   - Break `useBatches` into separate query, polling, and mutation primitives with explicit request ownership.
   - Fix current stale-state/race bugs in `DeviceSelector` and `TransformPreview` as part of that same cutover.
   - Preserve the current job-creation route structure while reducing cross-feature coupling.

3. **Reduce UI implementation sprawl in the highest-churn operator surfaces.**
   - Start with watch/export/auth/result detail surfaces where inline-style and component-size sprawl are highest.
   - Move repeated layout/color primitives into shared classes/tokens instead of duplicating inline style objects.
   - Use the refactor to shrink the largest Web files and keep future UX work localized.

