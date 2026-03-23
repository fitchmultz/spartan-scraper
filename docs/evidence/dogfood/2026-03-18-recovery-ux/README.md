# Recovery UX Dogfood — 2026-03-18

## Objective
Validate the in-product recovery UX in real degraded scenarios before moving on to cross-surface diagnostics parity.

## Environment
- Host OS: macOS
- Browser used for validation: Chrome driven via Playwright CLI and agent-browser
- Spartan commit: `233e86a`
- Web command: `make web-dev`
- Server commands:
  - Setup mode: `DATA_DIR=/tmp/spartan-dogfood/setup-legacy PI_ENABLED=true PI_NODE_BIN=/definitely-missing-node PI_BRIDGE_SCRIPT=/tmp/also-missing-bridge.js PROXY_POOL_FILE=/tmp/spartan-dogfood/setup-proxy.json USE_PLAYWRIGHT=1 go run ./cmd/spartan server`
  - Runtime degraded state: `DATA_DIR=/tmp/spartan-dogfood/runtime-ok PI_ENABLED=true PI_NODE_BIN=/definitely-missing-node PI_BRIDGE_SCRIPT=/tmp/also-missing-bridge.js PROXY_POOL_FILE=/tmp/spartan-dogfood/runtime-proxy.json USE_PLAYWRIGHT=1 PLAYWRIGHT_NODEJS_PATH=/definitely-missing-node go run ./cmd/spartan server`
- Data directories used:
  - Setup: `/tmp/spartan-dogfood/setup-legacy`
  - Runtime: `/tmp/spartan-dogfood/runtime-ok`

## Scenario Matrix

| Scenario | Expected | Observed | Status | Evidence |
| --- | --- | --- | --- | --- |
| Setup mode with legacy `jobs.db` | Setup panel visible with reset guidance | Passed after fixes; setup panel showed copy-ready reset guidance and no duplicate setup notice block | Pass | `01-setup-mode-all-actions.png` |
| Setup mode + degraded AI | AI degraded actions visible during setup mode | Passed after fixes; AI actions rendered beside setup recovery actions | Pass | `01-setup-mode-all-actions.png` |
| Setup mode + degraded proxy pool | Proxy recovery visible during setup mode | Passed after fixes; proxy-pool actions rendered in setup mode with operator wording | Pass | `01-setup-mode-all-actions.png` |
| Runtime degraded browser tooling | Browser recovery actions and one-click re-check visible | Passed; browser showed Playwright-specific guidance and one-click result | Pass | `05-browser-ai-runtime-degraded.png`, `09-one-click-browser-diagnostic.png` |
| Runtime degraded AI prerequisites | AI recovery actions and one-click re-check visible | Passed; AI showed actionable Node/bridge guidance and inline one-click result | Pass | `06-ai-runtime-degraded.png` |
| Runtime degraded proxy pool | Proxy recovery actions and one-click re-check visible | Passed after deleting the loaded proxy file and refreshing health | Pass | `07-proxy-runtime-degraded.png`, `10-proxy-one-click-result.png` |
| Copy actions | Copy buttons should show operator feedback and remain robust when Clipboard API is denied | Passed after fix; copy path now falls back cleanly instead of depending solely on `navigator.clipboard.writeText` | Pass | `08-copy-feedback.png` |
| One-click diagnostic actions | Run check → inline result → parent health refresh | Passed after fix; browser and proxy checks issued POSTs and were followed by `GET /healthz` refreshes | Pass | `09-one-click-browser-diagnostic.png`, `10-proxy-one-click-result.png`, `11-console-errors.txt` |
| External docs links | Open in a new tab and stay on the current Spartan tab | Passed; Playwright opened the browser guide in a second tab while leaving Spartan on the current tab | Pass | `10-external-link-browser-install-guide.png` |
| Route behavior during degraded states | Primary navigation should still work | Passed; navigation to `/settings/authoring` worked while the diagnostic panel remained actionable | Pass | Playwright tab/session log |

## Findings fixed during dogfood

### 1. Setup mode hid optional subsystem recovery and repeated setup copy
- Summary: Setup-mode `/healthz` only exposed the database/setup issue, which prevented setup-mode dogfood from showing degraded AI/proxy guidance. The panel also repeated the setup message in both the header and notices.
- Repro: Start setup mode with a legacy `jobs.db`, broken AI prerequisites, and proxy configuration. Before the fix, the panel did not surface the optional subsystem guidance cleanly.
- Fix: `internal/api/health.go` now includes browser/AI/proxy components in setup mode, and `web/src/components/SystemStatusPanel.tsx` suppresses duplicate setup notices while still showing degraded components.
- Evidence: `01-setup-mode-before-fix.png`, `01-setup-mode-all-actions.png`

### 2. Recovery commands were not copy-ready under `go run`
- Summary: Setup recovery actions used the transient Go build cache binary path, which was not operator-friendly.
- Repro: Start the server with `go run ./cmd/spartan server` in setup mode and inspect the reset command shown in the UI.
- Fix: `internal/cli/server/preflight.go` now normalizes `go run` launches to `go run ./cmd/spartan` and preserves stable relative binary paths like `./bin/spartan`.
- Evidence: setup-mode health payload before/after dogfood; `01-setup-mode-all-actions.png`

### 3. Copy buttons could silently fail when clipboard permissions were denied
- Summary: The copy path relied on `navigator.clipboard.writeText()` without fallback if that API rejected.
- Repro: In a browser context where Clipboard API permissions are denied, click a copy button.
- Fix: `web/src/components/SystemStatusPanel.tsx` now falls back to the legacy `execCommand("copy")` path when Clipboard API writes fail, and regression coverage was added.
- Evidence: `08-copy-feedback.png`

### 4. One-click degraded responses could recurse and crash the panel
- Summary: A degraded one-click result can include follow-up actions that themselves contain the same one-click action. Rendering those results recursively caused a browser `RangeError: Maximum call stack size exceeded`.
- Repro: Run a degraded one-click browser or AI check that returns follow-up actions, including the same `Re-check …` button.
- Fix: `web/src/components/SystemStatusPanel.tsx` now renders nested follow-up actions without recursively re-rendering prior inline results, and regression coverage was added.
- Evidence: pre-fix console stack trace referenced during dogfood; post-fix result screenshots `09-one-click-browser-diagnostic.png`, `10-proxy-one-click-result.png`

## Console / visual notes
- Post-fix console check showed no runtime errors or warnings from the recovery panel. See `11-console-errors.txt`.
- Wording reads clearly after the fixes. The remaining browser messaging is intentionally Playwright-specific when Chrome is present but Playwright prerequisites are broken.

## Final verification
- [x] Setup mode renders correctly with legacy jobs.db
- [x] All copy buttons work and show operator feedback
- [x] One-click diagnostic actions execute and show results
- [x] Health refresh updates the panel
- [x] Internal routes behave correctly in degraded states
- [x] External links open in new tabs
- [x] Wording is clear and actionable
- [x] Dogfood issues found were fixed
- [x] `make ci` passes
