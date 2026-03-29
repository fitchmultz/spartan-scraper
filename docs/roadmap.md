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

### RP-1: Land render-profile numeric validation hardening

Keep invalid persisted numeric input from being silently cleared, surface numeric range errors inline in `RenderProfileForm`, and keep the helper regression coverage in `settingsAuthoringForm.test.ts` aligned with the parser behavior.

Files: `settingsAuthoringForm.tsx`, `RenderProfileForm.tsx`, `RenderProfileEditor.test.tsx`, `settingsAuthoringForm.test.ts`

### RP-2: Consolidate template JSON codecs onto shared authoring helpers

Move template draft JSON parsing/formatting onto `settingsAuthoringForm.tsx`, update the template editor consumers to import the shared codecs, and keep the template-specific snapshot/payload helpers in `templateEditorUtils.ts` for now.

Files: `templateEditorUtils.ts`, `settingsAuthoringForm.tsx`, template consumer files

### RP-3: Prune `templateEditorUtils.ts` after the codec cutover

Delete dead exports that become unused after RP-2. If the file only keeps a few survivors, fold them into the owning template component and remove the helper file entirely.

Files: `templateEditorUtils.ts`, template workspace files

### App-1: Extract `useAppShellRouting` from `App.tsx`

Move pathname/history state, `parseRoute`, navigation helpers, popstate handling, canonical path enforcement, and promotion-seed state into `hooks/useAppShellRouting.ts`.

Files: new `web/src/hooks/useAppShellRouting.ts`, `App.tsx`

### App-2: Extract `useJobSubmissionActions` from `App.tsx`

Move job submission, cancel/delete, pending preset/submission state, and their supporting effects into `hooks/useJobSubmissionActions.ts`.

Files: new `web/src/hooks/useJobSubmissionActions.ts`, `App.tsx`

### App-3: Extract `useShellShortcuts` from `App.tsx`

Move keyboard navigation, assistant openers, route-help wiring, and onboarding route/action handlers into `hooks/useShellShortcuts.ts`.

Files: new `web/src/hooks/useShellShortcuts.ts`, `App.tsx`


## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
