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
| Settings AI affordances after fix | Render-profile and pipeline-JS AI actions should stop looking active/required when AI is unavailable, while manual authoring remains available | Pass after fix | `screenshots/25-settings-ai-disabled-buttons-closeup.png`, `screenshots/22-settings-ai-disabled-note.png` |

## Issue found and fixed

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

## Notes

- Fresh-start `/healthz`, CLI, and MCP all now frame optional-off states correctly:
  - AI: `disabled`
  - Proxy pool: `disabled`
  - Retention: disabled in its dedicated Settings/CLI flow without degrading core health
- The Web UI no longer shows the top-level System Status panel for a normal fresh start where nothing is actually wrong.
- After the settings AI-affordance fix, the first-run Settings route reads more coherently: manual paths stay primary, and optional AI help is clearly secondary.

## Final verification

- [x] Fresh-start health stayed `ok`
- [x] Optional-off proxy pool stayed `disabled` across Web/CLI/MCP
- [x] Retention stayed optional and actionable without looking broken
- [x] No fresh-start degraded system panel appeared for optional-off states
- [x] Settings AI affordances no longer feel required on first run
- [x] Regression tests were added for the settings AI-off state
- [x] `make ci` passes
