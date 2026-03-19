# Export outcome inspection dogfood

Date: 2026-03-18

## Goal

Manually validate the export outcome cutover across API, Web UI, CLI, and MCP, and confirm both direct exports and recurring export history now expose guided inspection and recovery details.

## Environment

- Backend: `DATA_DIR=<temp> ./bin/spartan server`
- Frontend: `make web-dev`
- Web URL: `http://localhost:5173`
- API URL: `http://127.0.0.1:8741`
- Real website exercised through the product: `https://example.com`

## Scenarios

| Scenario | Expected outcome | Result | Evidence |
| --- | --- | --- | --- |
| Direct export from the results route | Export drawer still downloads the artifact and now leaves behind a guided export outcome summary | Pass | `screenshots/01-job-results-export-drawer.png`, `screenshots/02-job-results-export-outcome.png` |
| Export schedule history in Automation | History modal shows guided status, narrative, artifact metadata, and recommended next steps instead of a raw table row | Pass | `screenshots/03-automation-exports-overview.png`, `screenshots/04-automation-export-history-modal.png` |
| API direct export inspection | `POST /v1/jobs/{id}/export`, `GET /v1/jobs/{id}/exports`, `GET /v1/exports/{id}`, and schedule history all return the new outcome envelope | Pass | `05-api-direct-export.json`, `06-api-job-export-history.json`, `07-api-export-outcome-get.json`, `04-api-export-schedule-history.json` |
| CLI export inspection | `spartan export` writes a file, persists an outcome, and can re-open job/schedule/export history from the same shared store | Pass | `08-cli-job-export.json`, `09-cli-export-inspect.json`, `10-cli-job-export-history.json`, `11-cli-schedule-export-history.json` |
| MCP export inspection | `job_export`, `job_export_history`, `export_outcome_get`, and `export_schedule_history` all return the same guided envelopes as REST | Pass | `12-mcp-job-export.json`, `13-mcp-job-export-history.json`, `14-mcp-export-outcome-get.json`, `15-mcp-schedule-export-history.json` |

## Notes

- The direct export workflow now feels inspectable instead of fire-and-forget. After download, the Web UI leaves the operator with a concrete export ID, destination, and follow-up actions.
- Export schedule history is much easier to reason about in-product: the modal shows the success narrative, artifact metadata, and reusable recovery actions without forcing the operator into CLI or logs first.
- API, CLI, and MCP all returned the same `export` / `exports` envelope shapes, which makes cross-surface automation and debugging more predictable.
- The fresh temporary workspace still showed the expected proxy-pool degradation banner because `proxy_pool.json` was intentionally absent, but export workflows remained usable and were not blocked.

## Issues found

None blocking during this validation pass.
