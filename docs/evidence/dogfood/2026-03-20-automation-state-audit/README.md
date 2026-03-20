# Automation workspace state audit

Date: 2026-03-20

## Goal

Audit the watches and exports automation workspaces for remaining empty, loading, zero-state, and notice-copy gaps after the promotion cutover, then harden the UI and regression coverage so operators never land on a blank or misleading automation surface.

## Environment

- Backend: `DATA_DIR=/tmp/spartan-automation-state-audit.BqJA9E PORT=8753 ./bin/spartan server`
- Delay proxy: local Python proxy on `127.0.0.1:8754` used only to hold watch/export list requests long enough for visible loading-state screenshots
- Frontend: `VITE_API_BASE_URL=http://127.0.0.1:8754 pnpm exec vite --port 5179`
- Web URL: `http://localhost:5179`
- API URL: `http://127.0.0.1:8753`
- Real website exercised for seeded automation data: `https://example.com`

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Watches first visit while the list request is still in flight | The watches route shows an explicit loading state instead of an empty table header shell | Pass | `screenshots/01-watches-initial-loading.png` |
| Watches with no saved rows after load completes | The route shows a guided empty state with clear next actions | Pass | `screenshots/02-watches-empty-state.png` |
| Exports first visit while the list request is still in flight | The export route shows an explicit loading state instead of a blank table | Pass | `screenshots/03-exports-initial-loading.png` |
| Exports with no saved rows after load completes | The route shows the guided empty state without the redundant always-on export paragraph | Pass | `screenshots/04-exports-empty-state.png` |
| History modal loading and empty branches | Watch/export history modals render structured product copy instead of bare loading text and ad-hoc empty paragraphs | Pass | `web/src/components/watches/WatchHistoryModal.test.tsx`, `web/src/components/export-schedules/ExportScheduleHistory.test.tsx` |
| Seeded automation data remains available for follow-up inspection work | A real watch baseline and a real export history record exist in the audit data set | Pass | `01-watch-list.json`, `02-watch-history.json`, `03-export-schedules.json`, `04-export-history.json`, `05-job-detail.json` |
| Full local CI gate after the audit fixes | All repo checks pass with the new manager/list/history coverage in place | Pass | `07-make-ci.txt` |

## Issues found and resolved in the same cut

1. **Initial watch/export loads could render blank table shells.**
   - Fix: initialize container loading state to true and teach both managers to show a dedicated loading notice when no rows have loaded yet.

2. **Export schedules repeated route guidance in a way that diluted the real empty state.**
   - Fix: removed the redundant always-visible paragraph so the route-level empty state owns the operator guidance.

3. **History modals looked unfinished in loading and empty branches.**
   - Fix: replaced bare `Loading...` and ad-hoc empty paragraphs with the shared `ActionEmptyState` presentation, including structured copy for watch detail loading.

4. **List component docs no longer matched real ownership.**
   - Fix: updated `WatchList` and `ExportScheduleList` comments to reflect that sorting is internal while empty/loading states remain manager-owned.

5. **Regression coverage missed the state branches most likely to rot.**
   - Fix: added manager, list, and history modal tests so loading, empty, sorted-row, and detail-loading behavior stay locked down.

## Notes

- The browser screenshots focus on the operator-visible route-level states that motivated this audit.
- A lightweight delay proxy was used only to make the initial loading states visible long enough to inspect. The persisted-data fixtures were seeded directly through the real API on `127.0.0.1:8753`.
- The next roadmap item should codify the post-promotion automation workspace path in deterministic regression coverage so these state fixes stay protected.
