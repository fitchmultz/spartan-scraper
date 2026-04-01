# 2026-04-01 Batch 3 jobs and results density cleanup

## Goal

Validate the Batch 3 density cutover: keep completed-job actions, promotion actions, and export controls easier to reach on `/jobs` and `/jobs/:id`, then confirm repeated route hopping across `/jobs`, `/jobs/:id`, `/templates`, and `/automation/*` feels direct on both desktop and mobile-width layouts.

## Environment

- Backend: `DATA_DIR=<temp> PORT=8759 ./bin/spartan server`
- Frontend: `DEV_API_PROXY_TARGET=http://127.0.0.1:8759 pnpm exec vite --host 127.0.0.1 --port 5179`
- Web URL: `http://127.0.0.1:5179`
- Desktop viewport: `1280×720`
- Mobile viewport: `390×844`
- Source URL used for the verification job: `https://example.com`
- Verified scrape job: `a106fb1f-0ebf-4268-9f69-b459bc5e4ef2`

## Flow covered

1. Create a fresh scrape job for `https://example.com` through the API-backed local environment.
2. Open `/jobs` and confirm recent completed work is reachable from a compact quick-access rail without dropping into the full lane stack first.
3. Open `/jobs/:id` and confirm the successful job exposes compact promote/export actions before the main reader.
4. Hop from the quick-action rail into `/templates`, `/automation/watches`, and `/automation/exports`.
5. Repeat the key `/jobs` and `/jobs/:id` checks at mobile width.

## Acceptance summary

| Surface | Expectation | Result | Evidence |
| --- | --- | --- | --- |
| Jobs dashboard (desktop) | Recent completed work is reachable before the filter and lane stack, with direct `View results` actions visible above the fold | Pass | `screenshots/01-jobs-desktop.png` |
| Job detail (desktop) | Promote/export actions sit above the reader without forcing the operator through the full promotion cards or export drawer first | Pass | `screenshots/02-job-detail-desktop.png` |
| Template hop | `Save as Template` still lands in the canonical `/templates` workspace from the compact quick-action rail | Pass | `screenshots/03-templates-desktop.png` |
| Automation hops | `Create Watch` and `Create Export Schedule` still seed the canonical automation drafts from the same quick-action rail | Pass | `screenshots/04-automation-watch-desktop.png`, `screenshots/05-automation-export-desktop.png` |
| Jobs dashboard (mobile) | The quick-access rail remains visible before the filter stack consumes the full screen height | Pass | `screenshots/06-jobs-mobile.png` |
| Job detail (mobile) | The compact operator-action block remains reachable early in the scroll and keeps promote/export buttons grouped predictably | Pass | `screenshots/07-job-detail-mobile.png`, `screenshots/08-automation-export-mobile.png` |
| Browser health | No browser errors during the pass; console output stayed limited to expected Vite/React development noise | Pass | `09-browser-console.txt`, `10-browser-errors.txt` |

## Notes

- `/jobs` now prioritizes return-to-results behavior by surfacing recent completed runs ahead of the denser filter and monitoring-lane controls.
- `/jobs/:id` now keeps the most common next steps in a compact operator-action rail, while the full promotion chooser remains available lower in the route for detailed carry-forward review.
- Mobile-width checks confirmed the same route-hop model still works without introducing a separate mobile-only flow.