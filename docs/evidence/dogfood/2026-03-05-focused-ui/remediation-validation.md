# Remediation Validation: Focused UI Dogfood Issues

Date: 2026-03-05
Source report: `docs/evidence/dogfood/2026-03-05-focused-ui/report.md`

## Status Summary
This patch set addresses all five issues found in the focused UI dogfood pass and adds regression tests for the affected hooks/components.

## Issue-by-Issue Status

### ISSUE-001 — Welcome modal after onboarding reset
**Status:** Fixed
- Removed automatic `isFirstLoad` timeout clear in `web/src/hooks/useOnboarding.ts`.
- `resetOnboarding()` now resets to a true welcome state instead of auto-starting the tour.
- Added hook tests in `web/src/hooks/useOnboarding.test.ts`.

### ISSUE-002 — `?` shortcut not opening help
**Status:** Fixed
- Updated shortcut matching to allow implicit Shift for `?` in `web/src/hooks/useKeyboard.ts`.
- Added tests in `web/src/hooks/useKeyboard.test.ts`.

### ISSUE-003 — Command palette focus not capturing typing
**Status:** Fixed
- Added explicit input ref + autofocus handoff on open in `web/src/components/CommandPalette.tsx`.

### ISSUE-004 — Onboarding cannot reach final step
**Status:** Fixed
- Added shared step-count source `web/src/lib/onboarding.ts`.
- Updated onboarding hook to use shared count in `web/src/hooks/useOnboarding.ts`.
- Updated Joyride callback handling to process `STEP_AFTER` and `TARGET_NOT_FOUND` using `ACTIONS/EVENTS/STATUS` in `web/src/components/OnboardingFlow.tsx`.
- Added regression tests in `web/src/components/OnboardingFlow.test.tsx`.

### ISSUE-005 — API outage WS churn / render loop instability
**Status:** Fixed
- Stabilized callback handling in `web/src/hooks/useWebSocket.ts` using callback refs without reconnect churn.
- Eliminated manual-disconnect double-callback behavior.
- Added regression tests in `web/src/hooks/useWebSocket.test.ts`.

## Additional UX Copy Alignment
To match actual shortcut behavior (non-modifier shortcuts are blocked while typing), copy was clarified in:
- `web/src/components/WelcomeModal.tsx`
- `web/src/components/OnboardingFlow.tsx`
- `web/src/components/KeyboardShortcutsHelp.tsx`

## Validation Run
- Local deterministic gate: `make ci` ✅
- Web tests: 18 files / 231 tests passing ✅

## Follow-up
A short live browser re-pass is recommended to append fresh post-fix screenshots/videos under this evidence folder.
