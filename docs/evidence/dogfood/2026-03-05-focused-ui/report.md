# Investigation: Focused UI Dogfood Pass (Spartan Scraper Web UI)

Date: 2026-03-05
Target: http://localhost:5173 (API: http://127.0.0.1:8741)
Session: `spartan-ui-dogfood`
Scope: Reviewer-critical UX paths (onboarding, command palette, shortcuts, API-failure behavior)

## Summary
I found 5 concrete UX issues with reproducible evidence (screenshots + videos). Two are critical for reviewer confidence:
1) onboarding tour cannot be completed reliably,
2) API outage triggers runaway console error spam plus repeated React max-depth errors.

## Symptoms
- First-visit onboarding welcome is not shown after clearing onboarding state.
- `?` shortcut does not open keyboard help.
- Command palette opens, but search focus is not reliably captured from active form fields.
- Onboarding flow drops the user out before the final completion step.
- Backend outage causes repeated WebSocket errors and "Maximum update depth exceeded" errors.

## Investigation Log

### Phase 1 – Initial Assessment
**Hypothesis:** onboarding/command-palette/shortcut surfaces have first-impression friction.  
**Findings:** confirmed by live UI pass with reproducible artifacts.  
**Evidence:**
- `docs/evidence/dogfood/2026-03-05-focused-ui/screenshots/initial.png`
- `docs/evidence/dogfood/2026-03-05-focused-ui/snapshot-initial.txt`

### Phase 2 – Context Builder + Oracle Triage
**Hypothesis:** likely failure hotspots are onboarding state, shortcut matching, command palette focus, and WS fallback behavior.  
**Findings:** context builder selected correct files; oracle hypotheses matched observed failures.  
**Evidence:** chat `ui-dogfood-investigation-B6FE3A` and selected files list.

### Phase 3 – Live Verification
**Hypothesis:** failures are reproducible in browser, not just code smell.  
**Findings:** all issues below reproduced directly in UI session.  
**Evidence:** per-issue screenshots/videos listed below.

## Confirmed Issues

### ISSUE-001 — First-visit welcome modal does not appear after state reset
**Severity:** High  
**Category:** UX / Onboarding  
**URL:** `http://localhost:5173/`  
**Repro Video:** `videos/issue-001-repro.webm`

**Repro steps**
1. Start from main page.  
   - `screenshots/issue-001-step-1.png`
2. Clear onboarding localStorage and reload.  
   - `screenshots/issue-001-step-2.png`
3. Observe no welcome modal despite fresh onboarding state.  
   - `screenshots/issue-001-result.png`
4. Confirm stored state is still uncompleted/unskipped.  
   - `issue-001-snapshot.txt`

**Expected:** Welcome modal appears for first-time state.  
**Actual:** Main UI loads directly; no welcome modal.

**Root-cause evidence**
- `web/src/hooks/useOnboarding.ts:192-196` clears `isFirstLoad` after 100ms.
- `web/src/hooks/useOnboarding.ts:277-282` `shouldShowWelcome` requires `isFirstLoad === true`.

---

### ISSUE-002 — `?` shortcut does not open keyboard shortcuts help
**Severity:** Medium  
**Category:** Accessibility / Discoverability  
**URL:** `http://localhost:5173/`  
**Repro Video:** `videos/issue-003-repro.webm`

**Repro steps**
1. Focus a non-input element on the page.  
   - `screenshots/issue-003-step-2.png`
2. Press `Shift+/` (`?` on US keyboard).  
   - `screenshots/issue-003-result.png`
3. Verify no keyboard help modal in accessibility snapshot.  
   - `issue-003-snapshot-2.txt`

**Expected:** `?` toggles keyboard shortcuts help modal.  
**Actual:** No help modal opens.

**Root-cause evidence**
- `web/src/hooks/useKeyboard.ts:70` default shortcut is `help: "?"`.
- `web/src/hooks/useKeyboard.ts:160-165` requires exact `shift` modifier match; `?` handling is not normalized, so matching is brittle.

---

### ISSUE-003 — Command palette opens, but typing still writes into active form field
**Severity:** High  
**Category:** Functional / UX  
**URL:** `http://localhost:5173/`  
**Repro Video:** `videos/issue-007-repro.webm`

**Repro steps**
1. Focus "Target URL" input in the scrape form.  
   - `screenshots/issue-007-step-1.png`
2. Open command palette with `⌘K`.  
   - `screenshots/issue-007-step-2.png`
3. Type `restart`.  
   - `screenshots/issue-007-step-3.png`
4. Confirm input value changed to `restart`.  
   - `issue-007-value2.txt`

**Expected:** Command palette search input captures typing.  
**Actual:** Typing goes to underlying form input.

**Root-cause evidence**
- `web/src/components/CommandPalette.tsx:284-288` command input has no explicit autofocus/focus handoff.
- `web/src/hooks/useKeyboard.ts:295-302` mod shortcuts are allowed while in inputs, but no focus transfer to palette input follows open.

---

### ISSUE-004 — Onboarding tour cannot reach final completion step
**Severity:** Critical  
**Category:** Functional / Onboarding  
**URL:** `http://localhost:5173/?showHelp=1`  
**Repro Video:** `videos/issue-005-repro.webm`

**Repro steps**
1. Open `?showHelp=1` to start tour.  
   - `screenshots/issue-005-showhelp-step-1.png`
2. Progress through steps (Step 2..7 visible).  
   - `screenshots/issue-005-step-4.png`
   - `screenshots/issue-005-step-5.png`
   - `screenshots/issue-005-step-6.png`
   - `screenshots/issue-005-step-7.png`
3. Click Next from Step 7.  
4. Observe tour controls disappear and final "You're All Set" completion step never appears.  
   - `screenshots/issue-005-step-8.png`
   - `issue-005-step-8.txt`

**Expected:** Final step appears with "Finish" and onboarding completion.  
**Actual:** Tour UI disappears; no completion action shown.

**Root-cause evidence**
- `web/src/hooks/useOnboarding.ts:71` `TOTAL_STEPS = 7`.
- `web/src/components/OnboardingFlow.tsx:33` `TOUR_STEPS` defines 8 steps (including final completion step).
- `web/src/components/OnboardingFlow.tsx:362-363` callback only increments on `step:after` complete; no `target_not_found` recovery path.

---

### ISSUE-005 — Backend outage causes runaway WS error spam + repeated React max-depth errors
**Severity:** Critical  
**Category:** Reliability / Console  
**URL:** `http://localhost:5173/` with API down  
**Repro Video:** `videos/issue-004-repro.webm`

**Repro steps**
1. Stop backend server.
2. Reload UI.  
   - `screenshots/issue-004-result.png`
3. Observe console and errors output.  
   - `issue-004-console.txt`
   - `issue-004-errors.txt`

**Expected:** graceful degraded state (polling/offline) with bounded retries and minimal console noise.  
**Actual:** repeated WS failures and many `Maximum update depth exceeded` errors.

**Root-cause evidence**
- `web/src/hooks/useAppData.ts:286-294` passes inline `onConnect`/`onDisconnect` callbacks into `useWebSocket`.
- `web/src/hooks/useWebSocket.ts:202-215` reconnect effect depends on `connect`; unstable callback identities can force reconnect churn.

---

## Eliminated Hypotheses
- **"Command palette is completely broken"** — eliminated. `⌘K` does open palette (`issue-002-snapshot-3.txt`, `screenshots/issue-007-step-2.png`).
- **"`?showHelp=1` is ignored"** — eliminated. It starts onboarding (`issue-006-location.txt`, `issue-006-grep.txt`).

## Recommendations
1. **Fix onboarding state contract**
   - Align total-step source of truth (use `TOUR_STEPS.length` instead of hardcoded `7`).
   - Remove auto-clear timer for welcome display or gate it after explicit user action.
2. **Fix command palette focus ownership**
   - Autofocus `Command.Input` when `isOpen` changes; enforce focus trap.
3. **Fix help shortcut normalization**
   - Normalize `?` as `shift+/` (or equivalent) in matcher and docs.
4. **Harden onboarding callback handling**
   - Handle `target_not_found` and move to next valid step with telemetry/logging.
5. **Stabilize WS fallback behavior under outage**
   - Memoize `onConnect`/`onDisconnect` callbacks passed to `useWebSocket`.
   - Bound retry logs and prevent render-loop cascades.

## Preventive Measures
- Add browser E2E smoke for onboarding happy path and completion (`?showHelp=1` path).
- Add keyboard shortcut integration test for `?` and `⌘K`/`Ctrl+K` with focus assertions.
- Add outage resilience test validating no React max-depth errors and bounded WS retry logs.
- Add release checklist gate: "No repeated console errors under API-down scenario (30s run)."
