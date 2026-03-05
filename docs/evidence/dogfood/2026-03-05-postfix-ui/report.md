# Investigation: Post-fix Live UI Re-dogfood (Spartan Scraper Web UI)

Date: 2026-03-05
Target: http://localhost:5173 (API: http://127.0.0.1:8741)
Primary Session: `spartan-ui-postfix`
Outage-isolation Session: `spartan-ui-postfix-outage`
Scope: Verify fixes for the 5 issues reported in `../2026-03-05-focused-ui/report.md`.

## Summary
All 5 previously reported UX/reliability regressions were retested live and passed.

- ✅ ISSUE-001 fixed: welcome modal appears on fresh onboarding state.
- ✅ ISSUE-002 fixed: `?` opens keyboard shortcuts help.
- ✅ ISSUE-003 fixed: command palette captures focus and typing from active form input context.
- ✅ ISSUE-004 fixed: onboarding reaches final completion step and can finish.
- ✅ ISSUE-005 fixed (critical portion): API outage no longer triggers `Maximum update depth exceeded`.

A small amount of expected outage noise remains (WebSocket and fetch failures while API is intentionally down), but no render-loop crash behavior was observed.

## Investigation Log

### Check 001 — Welcome modal appears after onboarding state reset
**Result:** PASS  
**Evidence:**
- `screenshots/check-001-welcome-after-reset.png`
- `check-001-snapshot.txt` (contains:
  - `button "🚀 Take a Quick Tour"`
  - `button "Skip for now — you can always restart the tour from the Command Palette"`)

### Check 002 — `?` opens keyboard shortcuts help
**Result:** PASS  
**Evidence:**
- `screenshots/check-002b-help-open.png`
- `check-002b-help-open-snapshot.txt` (help close control present)
- `check-002b-help-open.json`:
  - `hasKeyboardShortcutsText: true`

### Check 003 — Command palette focus ownership from active form input
**Result:** PASS  
**Evidence:**
- `check-003-after-metak.txt` (palette open; `combobox "Command palette" [expanded]`)
- `screenshots/check-003-command-palette-typed.png`
- `check-003-focus-values-2.json`:
  - active element is command input (`ariaLabel: "Search commands"`, value `restart`)
  - command input value is `restart`
  - original target field still retains `DOGFOOD_TARGET_SENTINEL`

### Check 004 — Onboarding can reach final step and finish
**Result:** PASS  
**Evidence:**
- `screenshots/check-004-tour-step1.png`
- `check-004-tour-advance-log.txt` (advances through Step 2..7 then final step visible)
- `screenshots/check-004-tour-final-step.png`
- `check-004-tour-final-check.json`:
  - `hasYouAreAllSet: true`
  - `hasFinishButton: true`
- `screenshots/check-004-tour-after-finish.png`

### Check 005 — API outage no longer causes max-depth render loop
**Result:** PASS (for regression target)  
**Evidence:**
- `screenshots/check-005-outage-isolated.png`
- `outage-isolated/console.txt`
- `outage-isolated/errors.txt`
- `outage-isolated/counts.txt`:
  - `max_update_depth_in_console=0`
  - `max_update_depth_in_errors=0`
  - `websocket_error_lines=1`
  - `failed_fetch_manager_lines=11`

## Conclusion
Post-fix live validation confirms the previously blocking onboarding/shortcut/focus regressions are resolved and the outage path no longer enters a React max-depth error loop. The bundle is ready to attach as remediation evidence for public-release review.
