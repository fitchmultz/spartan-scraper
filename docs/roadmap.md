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


### EU-1: Replace render-derived state and guarded initialization effects

Three effects either derive state from render inputs or seed defaults when async data arrives. Compute values during render when possible; otherwise reset or seed the owning subtree when the underlying identity changes.

| File:line | Current pattern | Fix |
|---|---|---|
| `VisualSelectorBuilder.tsx:188` | `setExpandedPaths(buildExpandedPaths(domTree))` | Reset preview-only tree state in the fetch-success path or by keying a preview subcomponent on the loaded page identity; keep the URL/runtime controls outside that keyed subtree |
| `useResultsSelectionState.ts:61` | `setTreeExpandedIds(buildDefaultExpandedTreeIds(treeNodes))` | Seed the tree expansion only once when the first tree appears; use a guarded initializer or a data-load reset, not render-time recomputation |
| `useResultsState.ts:119` | selected-result summary/confidence/evidence/citations sync | Derive the selected-result fields with `useMemo` from `resultItems[selectedResultIndex]`; keep only the selected-index clamp as an effect |

Reviewed and retained: `AICandidateDiffView` keeps user-controlled raw-JSON disclosure state, so it is not a derived-state misuse.

### EU-2: Replace prop-change reset effects with parent-level keys or direct resets

Three effects exist solely to reset local UI state when the owning identity changes. Key the component at the parent boundary or clear the state directly in the event/open path; do not key an inner wrapper.

| File:line | Resets on | Key strategy |
|---|---|---|
| `CommandPalette.tsx:142` | `isOpen` | Remount the palette from the shell when the open flag changes, or clear `search` directly in `onOpen`; do not key an inner wrapper |
| `JobSubmissionAssistantSection.tsx:164` | `activeTab` | Key the assistant section from `JobSubmissionContainer` by `activeTab` so the mode-specific preview state remounts cleanly |
| `ResultsAssistantSection.tsx:145` | `selectionResetKey` | Key the section from the results rail by job/selection identity so shape and refine drafts reset together |

### EU-3: Move own-state-change effects into event handlers or reducers

Two effects react to state transitions the component or hook itself initiated. The response belongs in the same setter, handler, or reducer action that caused the transition.

| File:line | Effect does | Fix |
|---|---|---|
| `useFormState.ts:410` | Clears `usePlaywright` when `headless` is off | Fold the reset into the `setHeadless` path or the shared form reducer so the illegal combination never exists |
| `RenderProfileForm.tsx:356` | Clears form error when draft changes | Clear the error in the same draft-mutating setters or reducer action instead of a separate effect |

### EU-4: Preserve legitimate external-system sync effects; revisit only if we adopt a query layer

These mount/route fetches are external-system synchronization, not misuse. Keep them in place for now.

| File:line | Why it stays |
|---|---|
| `useAppData.ts:449` | Bootstraps app-wide health/job/profile/schedule/template state from the server |
| `routes/AppRoutes.tsx:282` | Loads job results when a detail route mounts |
| `useTemplateDetailLoader.ts:103` | Fetches template detail on demand when the selected template changes |
| `useSettingsAuthoringShell.ts:198` | Loads the settings inventory on mount |
| `WatchContainer.tsx:258` | Refreshes the watch list |
| `ExportScheduleContainer.tsx:292` | Refreshes export schedules |
| `ChainContainer.tsx:105` | Refreshes chains |
| `WebhookDeliveryContainer.tsx:164` | Refreshes webhook deliveries |
| `useBatches.ts:635` | Initial batch fetch / polling refresh effect; keep it, and move the storage-load initializer to EU-5 |
| `RetentionStatusPanel.tsx:260` | Refreshes retention status |
| `ProxyPoolStatusPanel.tsx:80` | Refreshes proxy-pool status |

### EU-5: Replace storage-load mount effects with `useState` initializers

Four effects load values from `localStorage` on mount. A `useState(() => ...)` initializer does the same job without an extra render cycle.

| File:line | Loads | Fix |
|---|---|---|
| `useKeyboard.ts:219` | Stored keyboard shortcuts | Move into `useState(() => { ... stored; return { ...DEFAULT_SHORTCUTS, ...stored }; })` |
| `useTheme.ts:126` | Stored theme preference | Move into `useState(() => readStoredTheme() ?? 'system')` and call `applyTheme` in a single mount effect |
| `usePresets.ts:97` | Stored custom presets | Move into `useState(() => { try { return loadFromStorage(); } catch { return []; } })` |
| `useBatches.ts:643` | Last submitted batch notice | Load it in a `useState` initializer and keep the persistence effect for writes |

### EU-6: Convert async computation effects to explicit action triggers

One effect runs an async computation in response to state changes. Expose an explicit action and call it from the compare-selection path; if the diff tool remains open, rerun it when the compare job changes.

| File:line | Computes | Fix |
|---|---|---|
| `useResultsOperationsState.ts:158` | Result diff when both job IDs and `activeTool === "diff"` | Expose a `runDiff()` action and call it from `setCompareJobId` and the diff-tool open path; do not wait for a reactive effect to discover the work |

**Audit summary:** 108 `useEffect` calls in `web/src`. 24 are legitimate external-system subscriptions (DOM events, timers, WebSocket, focus/scroll management). ~14 are borderline-acceptable storage syncs. The confirmed cutover items above are the ones to eliminate in dependency order: EU-1 → EU-5 → EU-2 → EU-3 → EU-6, starting with the simplest derived-state fixes and ending with the async action trigger. EU-4 is intentionally retained because those fetches are external-system sync, not misuse.

Reviewed but retained: `BatchList`, `AIExportShapeAssistant`, `TemplatePreviewPane`, `App` route handoffs, `TransformPreview`/`ExportScheduleForm` AI-availability guards, `useOnboarding`, and the page-jump input sync effects are legitimate or borderline and stay out of this cutover list.

### TEST-1: Standardize the Vitest localStorage harness

Direct web Vitest runs should not depend on operators remembering `NODE_OPTIONS=--localstorage-file=.vitest-localstorage`. Move the localStorage file path into a repo-owned test entrypoint or wrapper so the supported local test command is warning-free and matches `make test` / CI.

| Files | Goal |
|---|---|
| `Makefile`, `web/package.json`, `web/scripts/*` | Centralize the Vitest launch path and inject the localStorage file option once, instead of requiring ad hoc shell env setup |

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
