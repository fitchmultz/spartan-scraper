# Saved automation inspection acceptance dogfood

Date: 2026-03-20

## Goal

Validate the first-save inspection loop for automation artifacts created from promotion so operators can trust what they just saved without leaving the canonical management surfaces.

## Environment

- Backend: `DATA_DIR=/tmp/spartan-saved-automation-inspection.0OZyXa PORT=8751 ./bin/spartan server`
- Frontend: `DEV_API_PROXY_TARGET=http://127.0.0.1:8751 pnpm exec vite --port 5177`
- Web URL: `http://localhost:5177`
- API URL: `http://127.0.0.1:8751`
- Real website exercised through the product: `https://example.com`

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Inspect the saved template from `/templates` | The promoted template persists as a reusable custom template and preview still renders the verified selector against the source page | Pass | `03-templates-list.json`, `03-template-detail.json`, `04-template-preview.json`, `screenshots/01-template-preview.png` |
| Run a manual watch check from `/automation/watches` and inspect saved history | The saved watch checks successfully, opens a trustworthy result summary, and the persisted history/detail view matches that result | Pass | `05-watch-list.json`, `06-watch-check.json`, `07-watch-history-page.json`, `08-watch-history-detail.json`, `screenshots/02-watch-check-result.png`, `screenshots/03-watch-history.png` |
| Inspect export schedule history from `/automation/exports` after a matching future job completes | The saved export schedule records a real export outcome with actionable next steps and artifact metadata | Pass | `09-export-schedules.json`, `10-export-history.json`, `screenshots/04-export-history.png` |
| Browser health during the inspection pass | No browser-visible errors or console noise appear while inspecting saved automation | Pass | `11-browser-console.txt`, `12-browser-errors.txt` |

## Issues found and resolved in the same cut

1. **Export history actions needed row-level feedback and stale-response protection.**
   - Symptom: the export route could feel inert while history loaded, and late responses could race against a newer selection.
   - Fix: added per-row loading affordances and request sequencing in `ExportScheduleManager`, plus regression coverage for stale response handling.
   - Validation: `screenshots/04-export-history.png`, `10-export-history.json`, `web/src/components/ExportScheduleManager.test.tsx`.

2. **Watch history needed the same stale async protection as export history.**
   - Symptom: a delayed history/detail response could repopulate modal state after the operator had already moved on.
   - Fix: added independent request sequencing for watch history pages and details, plus regression coverage.
   - Validation: `screenshots/03-watch-history.png`, `07-watch-history-page.json`, `08-watch-history-detail.json`, `web/src/components/WatchManager.test.tsx`.

3. **Watch promotion incorrectly carried visual diff on from non-browser scrape jobs.**
   - Symptom: a promoted watch inherited `screenshotEnabled=true` even when the source job ran without headless Chromium or Playwright, which made the first manual check fail instead of feeling trustworthy.
   - Fix: watch promotion now keeps screenshot-based visual diff off unless the source job already validated a browser-backed runtime, and the draft explains why in the unsupported carry-forward copy.
   - Validation: `05-watch-list.json`, `06-watch-check.json`, `web/src/lib/promotion.test.ts`.

## Notes

- The saved automation follow-up loop now feels coherent: save from promotion, then immediately inspect from the destination surface without hidden state or misleading defaults.
- Template preview, watch history, and export history all now reinforce the same product model: saved automation should expose persisted, operator-facing evidence instead of ephemeral one-off feedback.
