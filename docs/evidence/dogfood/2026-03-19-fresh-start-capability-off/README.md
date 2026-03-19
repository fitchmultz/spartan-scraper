# Fresh-start optional-capability-off dogfood

Date: 2026-03-19

## Goal

Validate the fresh-start operator experience across Web, CLI, and MCP with optional capabilities turned off, confirm those states stay quiet instead of degraded, and fix any remaining UX that makes optional features feel required.

## Environment

- Backend: `env -u PROXY_POOL_FILE -u PI_ENABLED -u PI_NODE_BIN -u PI_BRIDGE_SCRIPT -u RETENTION_ENABLED DATA_DIR=<fresh-temp> PORT=8741 ./bin/spartan server`
- Frontend: `pnpm exec vite --host 127.0.0.1 --port 5173`
- Web URL: `http://127.0.0.1:5173`
- Fresh data directory: temporary empty `DATA_DIR`
- Optional capabilities intentionally left off:
  - AI unconfigured
  - proxy pool unconfigured
  - retention disabled

## Scenario matrix

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Fresh `/healthz` on an empty workspace | Overall health stays `ok`; optional-off states stay `disabled` instead of `degraded` | Pass | `01-healthz.json` |
| Fresh Jobs route | First-run guidance stays focused on creating one working job; no degraded system panel appears just because optional features are off | Pass | `screenshots/01-jobs-fresh-start.png` |
| Fresh Settings route before fixes | Optional sections remain visible and quiet by default; no optional feature should feel required | Needed follow-up | `screenshots/04-settings-fresh-start.png` |
| CLI health and optional capability commands | CLI should report AI/proxy/retention as intentional optional states, not failures | Pass | `14-cli-health.txt`, `15-cli-proxy-diagnostic.txt`, `16-cli-proxy-status.txt`, `17-cli-retention-status.txt` |
| MCP health and proxy diagnostics | MCP should mirror the same `disabled` framing and translated actions | Pass | `18-mcp-health-status.json`, `19-mcp-proxy-diagnostic.json` |
| Settings AI affordances after fix | Render-profile and pipeline-JS AI actions should stop looking active/required when AI is unavailable, while manual authoring remains available | Pass after fix | `screenshots/25-settings-ai-disabled-buttons-closeup.png`, `screenshots/22-settings-ai-disabled-note.png`, `screenshots/26-settings-ai-disabled-followup.png` |
| One-click optional-off diagnostics | Re-check actions that return `disabled` should render as informational guidance, not warning-colored failure UI | Pass after fix | `web/src/components/CapabilityActionList.test.tsx` |
| Automation and results AI affordances | Export-schedule AI suggestion buttons and results-transform AI generation should disable cleanly when AI is off while manual editing stays available | Pass after fix | `web/src/components/export-schedules/ExportScheduleForm.test.tsx`, `web/src/components/TransformPreview.test.tsx` |
| Retention disabled guidance future-proofing | Disabled retention must stay quiet even if `/healthz` later starts surfacing a retention component | Pass after fix | `web/src/components/RetentionStatusPanel.test.tsx` |

## Issues found and fixed

### 1. Settings still presented AI authoring as if it were immediately usable on a fresh start

- **Severity:** medium
- **Surface:** Web UI / Settings
- **Why it mattered:** On a fresh workspace with AI intentionally unconfigured, Settings still showed active-looking `Generate with AI` / `Tune with AI` affordances inside Render Profiles and Pipeline JavaScript. That made an optional capability feel expected and undermined the new quiet-by-default model.
- **Repro:**
  1. Start Spartan with a fresh empty `DATA_DIR` and no AI env configured.
  2. Open `/settings`.
  3. Scroll to Render Profiles or Pipeline JavaScript.
  4. Observe AI affordances shown alongside manual authoring with no inline explanation of why AI is unavailable.
- **Fix:**
  - Passed the shared AI health component into the Settings render-profile and pipeline-JS editors.
  - Added explicit optional-AI guidance in both sections.
  - Disabled AI-only generation/tuning actions while keeping manual create/edit flows available.
  - Added regression coverage for both editors.
- **Evidence:**
  - Before: `screenshots/04-settings-fresh-start.png`
  - After: `screenshots/25-settings-ai-disabled-buttons-closeup.png`

### 2. Disabled optional checks and secondary AI helpers still had warning-toned or active-looking states

- **Severity:** medium
- **Surface:** Web UI / Settings, Automation, Results
- **Why it mattered:** Even after the first Settings fix, some optional-off flows still felt noisier than they should. One-click diagnostics rendered every non-`ok` result as warning-styled feedback, and some secondary AI helpers in automation/results still looked immediately usable despite AI being intentionally off.
- **Repro:**
  1. Start Spartan with a fresh empty `DATA_DIR` and no AI env configured.
  2. Open Settings and run an optional-off one-click check such as proxy-pool re-check.
  3. Open automation export scheduling or the results transform tool.
  4. Observe disabled optional states styled as warnings or active-looking AI launchers without matching capability-aware guidance.
- **Fix:**
  - Treated `disabled` diagnostic responses as informational instead of warning-style failures.
  - Centralized AI capability copy for web surfaces so disabled-by-choice and degraded AI states stay distinct.
  - Extended AI-off gating to export-schedule AI suggestion buttons and the results transform AI helper while preserving manual editing paths.
  - Future-proofed retention guidance so a disabled retention component cannot be relabeled as “needs attention”.
  - Added API, CLI, MCP, and web regression coverage for disabled optional states.
- **Evidence:**
  - `web/src/components/CapabilityActionList.test.tsx`
  - `web/src/components/export-schedules/ExportScheduleForm.test.tsx`
  - `web/src/components/TransformPreview.test.tsx`
  - `web/src/components/RetentionStatusPanel.test.tsx`
  - `internal/api/health_test.go`
  - `internal/api/diagnostics_test.go`
  - `internal/cli/server/health_test.go`
  - `internal/mcp/diagnostics_test.go`

## Notes

- Fresh-start `/healthz`, CLI, and MCP all now frame optional-off states correctly:
  - AI: `disabled`
  - Proxy pool: `disabled`
  - Retention: disabled in its dedicated Settings/CLI flow without degrading core health
- The Web UI no longer shows the top-level System Status panel for a normal fresh start where nothing is actually wrong.
- After the Settings and secondary-AI follow-up fixes, manual paths stay primary across Settings, automation export schedules, and results transforms.
- Disabled optional diagnostics now read as informational guidance instead of warning-colored failure states.

## Final verification

- [x] Fresh-start health stayed `ok`
- [x] Optional-off proxy pool stayed `disabled` across Web/CLI/MCP
- [x] Retention stayed optional and actionable without looking broken
- [x] No fresh-start degraded system panel appeared for optional-off states
- [x] Settings AI affordances no longer feel required on first run
- [x] Disabled optional diagnostics render as informational guidance instead of warnings
- [x] Automation export-schedule and results-transform AI affordances now disable cleanly when AI is off
- [x] Retention disabled guidance stays quiet even with future health-component input
- [x] Regression tests were added across Web, API, CLI, and MCP for disabled optional states
- [x] `make ci` passes
