# Promotion flow acceptance dogfood

Date: 2026-03-20

## Goal

Run a focused operator pass across `/jobs/:id`, `/templates`, `/automation/watches`, and `/automation/exports` to validate that a real successful job can be promoted into trustworthy destination drafts and then saved as reusable automation without re-entry friction.

## Environment

- Backend: `DATA_DIR=<temp> PORT=8751 ./bin/spartan server`
- Frontend: `DEV_API_PROXY_TARGET=http://127.0.0.1:8751 pnpm exec vite --port 5177`
- Web URL: `http://localhost:5177`
- API URL: `http://127.0.0.1:8751`
- Real website exercised through the product: `https://example.com`

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Submit a real scrape job with inline extraction and screenshots | A succeeded job detail route loads authoritative results and exposes the promotion panel | Pass | `01-submit-job.json`, `02-job-detail.json`, `screenshots/04-job-detail.png` |
| Promote to a template draft and save it | `/templates` opens a seeded draft with source context and persists a reusable custom template | Pass | `screenshots/05-template-draft.png`, `screenshots/09-template-saved.png`, `03-templates-list.json` |
| Promote to a watch draft and save it | `/automation/watches` opens a seeded draft, preserves the verified target/runtime settings, and persists a watch | Pass | `screenshots/06-watch-draft.png`, `screenshots/10-watch-saved.png`, `04-watch-list.json`, `screenshots/12-watch-saved-fixed.png` |
| Promote to an export schedule draft and save it | `/automation/exports` opens a seeded draft for future matching jobs, keeps the schedule semantics explicit, and persists a schedule | Pass | `screenshots/07-export-draft.png`, `screenshots/08-export-draft-fixed.png`, `screenshots/11-export-saved.png`, `05-export-schedules.json` |
| Browser health during the focused pass | No browser-visible errors or console noise during the validated promotion flow | Pass | `06-browser-errors.txt`, `07-browser-console.txt` |

## Issues found and resolved in the same cut

1. **Duplicate recurring-export promotion notices made the export route feel noisy and indirect.**
   - Symptom: the promoted export draft rendered the same source-job notice twice on `/automation/exports`.
   - Fix: removed the redundant manager-level notice so the form owns one clear promotion notice.
   - Evidence: before `screenshots/07-export-draft.png`; after `screenshots/08-export-draft-fixed.png`.

2. **Freshly created watches rendered Go zero timestamps as year 1 instead of “Never”.**
   - Symptom: a newly created promoted watch displayed `12/31/1, 4:15:11 PM` in the Last Checked column.
   - Fix: taught shared datetime formatting to treat Go zero timestamps as empty-state values and added regression coverage.
   - Evidence: before `screenshots/10-watch-saved.png`, API payload `04-watch-list.json`; after `screenshots/12-watch-saved-fixed.png`.

## Notes

- The promotion flow now feels complete end to end: one successful job can become a saved template, watch, and export schedule without re-entering the verified target or silently persisting hidden defaults.
- The destination notices do the right job after the export cleanup: they explain what carried forward, what still needs operator review, and how to return to the source job.
- Export promotion correctly frames automation as **future matching job exports**, not rerunning the source job.
- The watch list now presents a sensible empty-state timestamp for newly created watches, which keeps the first-save experience trustworthy.
