# Automation workspace failure recovery dogfood

Date: 2026-03-20

## Goal

Validate failed watch checks and failed export outcomes through the automation workspaces so operator guidance, route actions, and recovery steps remain trustworthy when automation fails.

## Environment

- Backend: `DATA_DIR=TMPDIR/spartan-failure-dogfood-final.yEYArD/data PORT=8744 MAX_CONCURRENCY=0 ./bin/spartan server`
- Frontend: `cd web && DEV_API_PROXY_TARGET=http://127.0.0.1:8744 pnpm exec vite --host 127.0.0.1 --port 4176`
- Watch fixture site: `python3 -m http.server 8767 --bind 127.0.0.1 -d TMPDIR/spartan-failure-dogfood-final.yEYArD/site`
- Web URL: `http://127.0.0.1:4176`
- API URL: `http://127.0.0.1:8744`

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Watch fetch failure (`http://127.0.0.1:1`) | Failed check persists, immediate modal and history modal agree, no diff rendered, recovery actions stay actionable | Pass | `01-watch-failed-check.json`, `02-watch-history-failed-page.json`, `03-watch-history-failed-detail.json`, `screenshots/01-watch-failed-result.png`, `screenshots/02-watch-failed-history.png` |
| Watch bad selector failure (`selector: [`) | Invalid selector fails instead of silently recording an empty baseline | Pass | `04-watch-bad-selector.json`, `screenshots/03-watch-bad-selector.png` |
| Direct export with no result file | Failed outcome persists with `category=result` and non-retryable guidance | Pass | `05-export-no-results.json` |
| Export transform failure | Failed outcome uses `category=transform` and offers retry-without-transform plus retry-as-JSONL guidance | Pass | `06-export-transform-failure.json`, `screenshots/05-export-network-history.png` |
| Export network/timeout failure | Retryable failure points back to `/automation/exports` with route-aware actions | Pass | `07-export-network-failure.json`, `screenshots/05-export-network-history.png` |
| Mixed success/failure pagination | History pagination stays correct across mixed outcome states | Pass | `08-export-history-page-1.json`, `09-export-history-page-2.json`, `screenshots/06-export-history-pagination.png` |
| Browser health | No console errors or browser-reported errors while inspecting failure states | Pass | `10-browser-console.txt`, `11-browser-errors.txt` |
| Full local CI | `make ci` passes after fixes | Pass | `12-make-ci.txt` |

## Notes

- The watch selector regression was real before this cutover: an invalid selector such as `[` previously produced an empty baseline instead of a failed check.
- The export history workspace now uses the shared capability action renderer, so route actions render as navigation buttons while command actions render explicit copy affordances.
- Direct export failure persistence now records missing-result failures in canonical export history instead of returning early without an outcome record.
