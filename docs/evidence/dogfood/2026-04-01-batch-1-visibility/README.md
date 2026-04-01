# 2026-04-01 Batch 1 visibility acceptance

## Goal

Run the remaining Batch 1 manual acceptance pass at a 1280×720 viewport and confirm the first-run operator path keeps a visible next action across:

- job detail → export chooser
- promoted template authoring
- watch creation
- export schedule creation

## Environment

- Backend: `DATA_DIR=<temp> PORT=8758 ./bin/spartan server`
- Frontend: `DEV_API_PROXY_TARGET=http://127.0.0.1:8758 pnpm exec vite --host 127.0.0.1 --port 5178`
- Web URL: `http://127.0.0.1:5178`
- Viewport: `1280×720`
- Source URL exercised through the product: `https://example.com`

## Visual verification

| Surface | Expectation | Result | Evidence |
| --- | --- | --- | --- |
| New job review | Submit remains visible without hunting below the fold on the guided create-job path | Pass | `screenshots/03-job-review.png`, `screenshots/04-jobs-after-submit.png` |
| Job detail → export chooser | Export drawer exposes an immediate handoff action above the format grid | Pass | `screenshots/05-job-detail.png`, `screenshots/06-job-detail-export.png` |
| Promoted template authoring | The promoted `/templates` workspace makes the next authoring action visible immediately from the seeded draft surface | Pass | `screenshots/07-template-draft.png` |
| Watch creation | The promoted watch dialog keeps the seeded context and visible create action in the initial viewport, then saves cleanly | Pass | `screenshots/08-watch-draft.png`, `screenshots/09-watch-saved.png` |
| Export schedule creation | The promoted export-schedule dialog keeps the seeded context and visible create action in the initial viewport, then saves cleanly | Pass | `screenshots/10-export-draft.png`, `screenshots/11-export-saved.png` |
| Browser health during the pass | No browser errors; only expected Vite/React dev-mode informational console output | Pass | `12-browser-console.txt`, `13-browser-errors.txt` |

## Notes

- Batch 1 acceptance passed without additional product changes during this run.
- The job-detail export chooser now clearly surfaces immediate `Export JSONL now` / `Export CSV now` actions in the first visible region.
- Promoted watch and export schedule flows both saved successfully from their destination surfaces at the target laptop-height viewport.
- Template promotion remained actionable from the initial viewport through the seeded workspace controls and visible draft context; deeper blank-draft authoring guidance remains Batch 2 work, not a Batch 1 blocker.
