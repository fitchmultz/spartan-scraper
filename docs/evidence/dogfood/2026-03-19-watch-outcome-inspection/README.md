# Watch outcome inspection dogfood

Date: 2026-03-19

## Goal

Visually validate the new watch history inspection flow in the Automation Web UI after persisting manual watch checks through the shared history contract.

## Environment

- Backend: `DATA_DIR=<temp> ./bin/spartan server`
- Frontend: `make web-dev` on `http://localhost:5174`
- Watched page fixture: `python3 -m http.server 8765 --bind 127.0.0.1 --directory <temp-site>`
- Real product route exercised: `http://localhost:5174/automation/watches`

## Scenario

1. Create a watch for the local fixture page.
2. Run an initial manual check to establish the baseline.
3. Change the fixture page content and run another manual check so history captures a changed outcome.
4. Open the watch history modal from the Watches table.
5. Confirm the modal shows:
   - the newest changed record first,
   - prior unchanged/baseline checks in the left-hand timeline,
   - diff text for the selected changed check,
   - guided next-step actions in the detail pane.

## Result

Pass.

The modal rendered the expected stacked history list and detail pane, and the selected changed check showed the saved diff plus reusable CLI/Web follow-up actions.

## Evidence

- Screenshot: `output/playwright/watch-outcome-inspection-2026-03-19/watch-history-modal.png`
