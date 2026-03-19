# 2026-03-19 Docs and onboarding optional-capability alignment

## Goal

Align operator-facing docs, environment comments, onboarding, and in-app help so Spartan consistently presents AI, proxy pooling, and retention as optional, off-by-default capabilities instead of prerequisites or failures.

## What changed

- Updated root and operator docs to state that the default first run works without AI, proxy pooling, or retention.
- Reworded `.env` and `.env.example` comments for proxy, retention, and pi-backed AI settings so they read as enable-when-needed toggles.
- Tightened onboarding, route-help, and Settings copy so optional capabilities feel informational and available later, not required up front.
- Refined shared optional-capability UI copy for proxy pooling, retention, system status, and AI-unavailable notices to preserve the disabled-by-choice vs degraded distinction.
- Removed the completed docs/onboarding alignment item from `docs/roadmap.md` and recorded the resolution in `docs/specs/README.md` plus `docs/specs/web-ui-ux-audit.md`.

## Visual verification

Validated against a fresh local runtime with AI, proxy pooling, and retention left off by default.

| Surface | Expectation | Result | Evidence |
| --- | --- | --- | --- |
| Jobs first-run nudge | First-run copy should say no AI/proxy/retention setup is required | Pass | `screenshots/01-jobs-nudge-optional-capabilities.png` |
| Settings overview | Settings should describe optional capabilities as enable-later controls, not required setup | Pass | `screenshots/02-settings-optional-capabilities.png` |
| Templates route help | Route help should frame AI assistance as optional workflow help rather than a prerequisite | Pass | `screenshots/03-ai-unavailable-optional.png` |

## Automated verification

- `cd web && pnpm exec vitest run src/components/OnboardingNudge.test.tsx src/components/OnboardingFlow.test.tsx src/components/ProxyPoolStatusPanel.test.tsx src/components/RetentionStatusPanel.test.tsx src/components/render-profiles/RenderProfileEditor.test.tsx src/components/pipeline-js/PipelineJSEditor.test.tsx src/components/ai-assistant/__tests__/AIAssistantPanel.test.tsx`
- `make ci`

## Notes

- `docs/README.md` stayed unchanged after audit because it already routed operators through a value-first path without setup-first optional-capability language.
- This pass standardized narrative only. Degraded-state recovery guidance remains explicit when a capability is enabled and actually failing.
