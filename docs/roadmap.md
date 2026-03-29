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

### RP-1: Land render-profile numeric validation

Uncommitted work already in tree: `parseOptionalNumber` throws on invalid input, `RenderProfileForm` wires label errors, editor test covers invalid draft. Commit as-is, then verify no other numeric fields in `RenderProfileForm` silently swallow bad input. Add a shared-helper unit test for `parseOptionalNumber` edge cases (negative, zero, scientific notation) if the existing test doesn't cover them.

Files: `settingsAuthoringForm.tsx`, `RenderProfileForm.tsx`, `RenderProfileEditor.test.tsx`, `settingsAuthoringForm.test.ts`

### RP-2: Consolidate `parseJSONInput` onto shared authoring helpers

`templateEditorUtils.ts` has its own `parseJSONInput` and `formatJSON` that duplicate `parseOptionalJSONObject` and `formatOptionalJSON` from `settingsAuthoringForm.tsx`. Re-export the shared versions from `settingsAuthoringForm`, update all 7 template consumers to import from the shared hub, then delete the duplicates from `templateEditorUtils`. Keep template-specific helpers (`buildTemplateSnapshot`, `buildTemplatePayload`, `ruleKey`, draft helpers) in `templateEditorUtils` — they have no shared-authoring equivalent.

Files: `templateEditorUtils.ts`, `settingsAuthoringForm.tsx`, 7 template consumer files

### RP-3: Delete unused template editor helpers

After RP-2, audit `templateEditorUtils.ts` for any remaining functions that are no longer imported. Delete dead exports. If the file drops below ~80 lines of real code, fold the survivors into the template workspace component that uses them and delete the file entirely.

Files: `templateEditorUtils.ts`, possibly template workspace files

### App-1: Extract `useAppShellRouting` from `App.tsx`

Pull pathname state, `parseRoute`, `navigate`, popstate listener, automation/settings canonical-path effects, `handleNavigate`, `handlePaletteNavigate`, and `handleViewResults` into `hooks/useAppShellRouting.ts`. The hook takes `selectedJobId` and `persistJobsViewState` as inputs and returns `{ pathname, route, navigate, handleNavigate, handlePaletteNavigate, handleViewResults }`. `App.tsx` becomes a consumer, not the owner of routing.

Files: new `web/src/hooks/useAppShellRouting.ts`, `App.tsx`

### App-2: Extract `useJobSubmissionActions` from `App.tsx`

Pull `handleSubmitScrape`, `handleSubmitCrawl`, `handleSubmitResearch`, `cancelJob`, `deleteJob`, `pendingPreset`/`pendingSubmission` state, `handleSubmitForm`, `handleSelectPreset`, and their supporting effects into `hooks/useJobSubmissionActions.ts`. The hook takes `{ toast, navigate, appData, formState, jobSubmissionRef }` and returns the action callbacks. Remove `postV1Scrape`/`postV1Crawl`/`postV1Research`/`deleteV1JobsById` imports from `App.tsx`.

Files: new `web/src/hooks/useJobSubmissionActions.ts`, `App.tsx`

### App-3: Extract `useShellShortcuts` from `App.tsx`

Pull `openJobAssistant`, `openTemplateAssistant`, `getCurrentConfig`, `getCurrentUrl`, keyboard-navigate listener, onboarding route-change/action handlers, and `routeHelpProps` memo into `hooks/useShellShortcuts.ts`. The hook takes `{ aiAssistant, formState, navigate, route, activeTab, jobSubmissionRef, shortcuts, isMac }` and returns the shortcut-triggered actions and help props.

Files: new `web/src/hooks/useShellShortcuts.ts`, `App.tsx`

### App-4: Remove `clearPromotionSeed` history-state coupling from `App.tsx`

After App-1, the promotion-seed read/clear logic (`navigationState.promotionSeed`, `clearPromotionSeed`) can move into `useAppShellRouting` or a small `usePromotionSeed` helper inside the routing hook. This removes the last piece of history-state parsing from `App.tsx` proper. Verify `TemplatesRoute` and `AutomationRoute` still receive the seed via props.

Files: `useAppShellRouting.ts`, `App.tsx`

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
