# 2026-03-19 Settings first-run UX polish

## Goal

Make the Settings route feel calm and coherent on fresh installs by replacing scattered empty-state noise with one first-run narrative and matching optional-capability framing across all Settings panels.

## What changed

- Added a shared first-run `SettingsOverviewPanel` that explains when each major Settings capability matters instead of implying everything needs setup immediately.
- Removed the extra `Nothing needs maintenance yet` inventory banner so the auth/schedule/crawl sections stay focused on their own purpose.
- Cut the auth, schedule, and crawl-state empty cards down to one targeted next step each.
- Reworked render-profile and pipeline-JS panels to use the shared empty-state presentation, clearer section descriptions, and first-run copy aligned with optional runtime/page hooks.
- Lifted render-profile and pipeline-script inventory counts to the app shell so the overview only appears during a real pristine first-run state.
- Updated Settings route and onboarding copy to reinforce the “configure reuse later, start with a real job now” model.

## Visual verification

Validated on a fresh local data directory with optional subsystems off by default.

| Surface | Expectation | Result | Evidence |
| --- | --- | --- | --- |
| Settings first-run landing state | One calm overview should orient operators before section-level empty states begin | Pass | `screenshots/01-settings-overview.png` |
| Render profiles / pipeline JS empty states | Optional runtime and page-hook editors should explain manual first-run usage without looking broken | Pass | `screenshots/01-settings-overview.png` |
| Proxy pool / retention coexistence | Optional subsystem guidance should still sit naturally beside the new first-run narrative | Pass | `screenshots/01-settings-overview.png` |

## Automated verification

- `cd web && pnpm exec vitest run src/components/SettingsOverviewPanel.test.tsx src/components/InfoSections.test.tsx src/components/render-profiles/RenderProfileEditor.test.tsx src/components/pipeline-js/PipelineJSEditor.test.tsx`
- `make ci`

## Notes

- The first-run overview intentionally disappears once any reusable Settings inventory exists or an optional subsystem is no longer in a quiet disabled/ok state.
- Route-level onboarding and header copy now match the same defer-configuration-until-needed narrative.
