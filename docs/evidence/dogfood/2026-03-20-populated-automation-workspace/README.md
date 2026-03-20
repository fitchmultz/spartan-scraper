# Populated automation workspace acceptance dogfood

Date: 2026-03-20

## Goal

Validate the watches and exports automation workspaces with real saved rows, browser-triggered manual history entry points, and persisted history modals so operators can trust the populated save-and-inspect path without relying on any artificial delay harness.

## Environment

- Backend: `DATA_DIR=/tmp/spartan-populated-automation-workspace.mqLAGY PORT=8755 ./bin/spartan server`
- Frontend: `DEV_API_PROXY_TARGET=http://127.0.0.1:8755 pnpm exec vite --port 5180`
- Web URL: `http://localhost:5180`
- API URL: `http://127.0.0.1:8755`
- Monitored and exported target: `https://example.com`

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Watches workspace renders populated saved rows on direct `/automation/watches` load | The watch list shows the saved watch immediately with no empty-state fallback | Pass | `01-watch-list.json`, `screenshots/01-watches-list.png` |
| Manual watch check opens the immediate inspection modal and pivots into persisted history | The browser-triggered check shows `CheckResultModal`, then `View history` lands on the matching persisted check in `WatchHistoryModal` | Pass | `02-watch-check.json`, `03-watch-history-page-1.json`, `04-watch-history-detail.json`, `screenshots/02-watch-check-result.png`, `screenshots/03-watch-history-page-1.png` |
| Watch history pagination works with authoritative persisted rows | Page 2 of watch history loads correctly and older persisted checks can still drive the detail panel | Pass | `05-watch-history-page-2.json`, `screenshots/04-watch-history-page-2.png` |
| Export schedules workspace renders populated saved rows on direct `/automation/exports` load | The export schedule list shows the saved schedule and its filters/destination immediately | Pass | `06-export-schedules.json`, `screenshots/05-exports-list.png` |
| Export schedule history pagination works with authoritative export outcomes | The export history modal shows page 1 and page 2 correctly for the 11 seeded outcomes | Pass | `07-export-history-page-1.json`, `08-export-history-page-2.json`, `screenshots/06-export-history-page-1.png`, `screenshots/07-export-history-page-2.png` |
| Recommended automation links target the correct sub-route after the automation hub split | Watch actions point to `/automation/watches` and export actions point to `/automation/exports` instead of the legacy bare `/automation` route | Pass | `02-watch-check.json`, `03-watch-history-page-1.json`, `07-export-history-page-1.json`, `internal/api/watch_history_test.go`, `internal/api/export_schedules_test.go` |
| Browser health during the populated-data pass | No browser console noise or browser error entries appear on the final direct route loads | Pass | `09-browser-console.txt`, `10-browser-errors.txt` |
| Full local CI gate after the acceptance fixes | The full repo gate stays green after the route-action fix and evidence refresh | Pass | `11-make-ci.txt` |

## Issues found and resolved in the same cut

1. **Automation recommended-action links still targeted the legacy bare `/automation` route.**
   - Symptom: watch and export inspection actions landed on the default batches section after the automation sub-route cutover instead of returning operators to the relevant watch/export workspace.
   - Fix: updated backend-produced `RecommendedAction.value` strings to `/automation/watches` and `/automation/exports`, then added backend regression assertions so history payloads cannot drift back.
   - Validation: `02-watch-check.json`, `03-watch-history-page-1.json`, `07-export-history-page-1.json`, `internal/api/watch_history_test.go`, `internal/api/export_schedules_test.go`.

## Notes

- This pass used a fresh isolated `DATA_DIR` and the normal same-origin Vite proxy path. No delay proxy or artificial loading harness was used.
- Seeded authoritative data totals for the final pass were: 1 saved watch, 12 persisted watch checks after the browser-triggered check, 1 export schedule, and 11 persisted export outcomes.
- `02-watch-check.json` and `04-watch-history-detail.json` intentionally capture the same browser-triggered check through the shared `WatchCheckInspectionResponse` contract so the immediate and persisted inspection views can be compared with the same check ID.
