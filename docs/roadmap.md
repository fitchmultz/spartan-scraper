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

## Audit-Derived Follow-up Work (2026-03-30)

Audit snapshot: 156 non-test code files exceed 300 lines, and the current Go/TS heuristics found roughly 393 functions over 50 LOC.

### Cleanup Batch 4 — Large test surface trimming and fixture reuse

Why: Some of the biggest files in the repo are scenario-heavy tests, and trimming them will make failures easier to localize without weakening coverage.

- Split oversized test files into scenario-focused suites with shared builders/fixtures where that reduces duplication and improves failure locality.
- Prioritize the largest hand-maintained test hotspots from the audit sample, including `web/src/components/__tests__/FreshStartOperatorFlow.test.tsx`, `web/src/components/templates/__tests__/TemplateManager.test.tsx`, `internal/cli/manage/auth_oauth_test.go`, and `internal/cli/batch/batch_test.go`.
- Keep end-to-end assertions intact; this batch is about readability, fixture reuse, and lower-churn maintenance rather than changing test scope.

