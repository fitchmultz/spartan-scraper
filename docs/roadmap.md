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

### RP-3: Prune `templateEditorUtils.ts` after the codec cutover

Delete any remaining dead exports and fold the surviving helpers into the owning template components.

Files: `web/src/components/templates/templateEditorUtils.ts`, template workspace files

### EU-1: Remove render-derived state from results and visual-selector components

Derive preview state during render or reset the owning subtree when the loaded page identity changes.

Files: `web/src/components/VisualSelectorBuilder.tsx`, `web/src/components/results-explorer/useResultsSelectionState.ts`, `web/src/hooks/useResultsState.ts`

### EU-5: Replace storage-load mount effects with `useState` initializers

Load stored keyboard shortcuts, theme, presets, and last-submitted batch notice in initializers instead of mount effects.

Files: `web/src/hooks/useKeyboard.ts`, `web/src/hooks/useTheme.ts`, `web/src/hooks/usePresets.ts`, `web/src/hooks/useBatches.ts`

### EU-2: Remove prop-change reset effects from shell assistants

Key or clear the owning component for `CommandPalette`, `JobSubmissionAssistantSection`, and `ResultsAssistantSection`.

Files: `web/src/components/CommandPalette.tsx`, `web/src/components/ai-assistant/JobSubmissionAssistantSection.tsx`, `web/src/components/ai-assistant/ResultsAssistantSection.tsx`

### EU-3: Fold self-resetting effects into their setters or reducers

Remove the `usePlaywright` reset in `useFormState` and the render-profile draft-error reset in `RenderProfileForm` by handling the state transition at the source.

Files: `web/src/hooks/useFormState.ts`, `web/src/components/render-profiles/RenderProfileForm.tsx`

### EU-6: Replace diff recomputation with an explicit action

Expose `runDiff()` and call it from compare-selection and diff-tool entry points instead of relying on a reactive effect.

Files: `web/src/components/results-explorer/useResultsOperationsState.ts`

### App-1: Extract `useAppShellRouting` from `App.tsx`

Move pathname/history state, route parsing, navigation helpers, popstate handling, canonical path enforcement, and promotion-seed state into the hook.

Files: `web/src/hooks/useAppShellRouting.ts`, `web/src/App.tsx`

### App-2: Extract `useJobSubmissionActions` from `App.tsx`

Move job submission, cancel/delete, pending preset/submission state, and supporting effects into the hook.

Files: `web/src/hooks/useJobSubmissionActions.ts`, `web/src/App.tsx`

### App-3: Extract `useShellShortcuts` from `App.tsx`

Move keyboard navigation, assistant openers, route-help wiring, and onboarding handlers into the hook.

Files: `web/src/hooks/useShellShortcuts.ts`, `web/src/App.tsx`

### TEST-1: Standardize the Vitest localStorage harness

Put the localStorage file path behind a repo-owned Vitest entrypoint so direct web runs are warning-free and match CI.

Files: `Makefile`, `web/package.json`, `web/vitest.config.ts`

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
