# Operator Guide

## Deployment Model

- Supported deployment model: single node, trusted operator, local disk, SQLite.
- Default data root: `.data`.
- Job artifacts live under `.data/jobs/<job-id>/`.
- Each completed or failed job writes `.data/jobs/<job-id>/manifest.json`.

## Start Server

```bash
DATA_DIR=.data spartan server
```

## Canonical Diagnostics Workflow

Use the same recovery model everywhere:

```bash
spartan health
spartan health --check browser
spartan health --check ai
spartan health --check proxy_pool
spartan proxy-pool status
spartan retention status
```

- `spartan health` shows structured setup, runtime, component, and config notices.
- `spartan health --check ...` runs the same read-only re-checks surfaced by the Web UI.
- `spartan proxy-pool status` and `spartan retention status` now lead with capability-aware guidance before the low-level runtime/config details.
- If the API server is already running, the CLI reads `/healthz` and the diagnostic endpoints directly.
- If the API server is offline, the CLI falls back to local checks instead of hiding recovery guidance.

## Export Outcome Inspection

Direct exports and recurring export schedules now share one persisted outcome history.

Canonical operator workflow:

```bash
spartan export --job-id <job-id> --format json
spartan export --inspect-id <export-id>
spartan export --history-job-id <job-id>
spartan export-schedule history --id <schedule-id>
```

Cross-surface expectations:

- `POST /v1/jobs/{id}/export` returns `{ export }` with inline artifact content and guided recovery actions.
- `GET /v1/jobs/{id}/exports`, `GET /v1/exports/{id}`, and `GET /v1/export-schedules/{id}/history` all return the same outcome envelope shape.
- `spartan mcp` exposes the same model through `job_export`, `job_export_history`, `export_outcome_get`, and `export_schedule_history`.
- The Web UI now surfaces the same titles, messages, artifact metadata, failure context, and recommended next steps inside export history and direct-export flows.

## Watch Outcome Inspection

Manual and scheduled watch checks now share one persisted history.

Canonical operator workflow:

```bash
spartan watch check <watch-id>
spartan watch history <watch-id>
spartan watch history <watch-id> --check-id <check-id>
```

Cross-surface expectations:

- `GET /v1/watch` and MCP `watch_list` use the same paginated `{ watches, total, limit, offset }` contract.
- `POST /v1/watch/{id}/check` persists and returns the canonical `{ check }` inspection envelope immediately.
- `GET /v1/watch/{id}/history` and `GET /v1/watch/{id}/history/{checkId}` return the same guided inspection envelopes used by the Web UI and MCP.
- Historical artifact snapshots live under `/v1/watch/{id}/history/{checkId}/artifacts/{artifactKind}` so older checks stay inspectable after later runs rotate the latest watch artifacts.
- `spartan mcp` exposes the same model through `watch_check`, `watch_check_history`, and `watch_check_get`.
- The Web UI now lets operators jump from an immediate manual check into the saved history modal without leaving Automation.

## Setup Recovery

If startup blocks on legacy or unsupported persisted state, Spartan now stays in setup mode and keeps diagnostics available.

Canonical recovery path:

```bash
spartan reset-data
spartan server
```

That archives the current data directory to `output/cutover/` and recreates `DATA_DIR` for a fresh Balanced 1.0 store.

Alternative path:

```bash
DATA_DIR=/path/to/new-empty-dir spartan server
```

## MCP Diagnostics

`spartan mcp` now exposes diagnostics parity as well:

- `health_status`
- `diagnostic_check` with `component: browser | ai | proxy_pool`

In setup mode, MCP still starts but only exposes those diagnostics tools until recovery is completed.

Smoke examples:

```bash
printf '%s\n' '{"id":1,"method":"tools/call","params":{"name":"health_status","arguments":{}}}' | spartan mcp
printf '%s\n' '{"id":2,"method":"tools/call","params":{"name":"diagnostic_check","arguments":{"component":"browser"}}}' | spartan mcp
```

## Runtime Expectations

- Non-loopback API binds require API-key protection.
- Browser engines are local process dependencies; keep Chromium/Playwright availability aligned with your workload.
- Backup and restore must include the SQLite database and the full `.data/jobs/` tree.

## Backup And Restore

```bash
spartan backup create -o ./backups
spartan restore --from ./backups/spartan-backup-20260311-120000.tar.gz --dry-run
```

## Manifest Inspection

```bash
cat .data/jobs/<job-id>/manifest.json
```

The manifest is the canonical local index for a finished job’s files, spec hash, checksums, and selected engine.

## Storage Reset

- If startup reports a legacy storage schema, stop the server and run `spartan reset-data`.
- That command archives the full existing data directory to `output/cutover/` by default and recreates `DATA_DIR`.
- Start the server again after the reset completes.
- The same rule now applies to `schedules.json`: the retained 1.0 build only accepts typed V1 schedule specs on disk, not legacy `params`. The operator-facing API request shape is `kind` + `intervalSeconds` + `request`.
