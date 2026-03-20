# Operator Guide

## Deployment Model

- Supported deployment model: single node, trusted operator, local disk, SQLite.
- Default data root: `.data`.
- Job artifacts live under `.data/jobs/<job-id>/`.
- Each completed or failed job writes `.data/jobs/<job-id>/manifest.json`.
- AI, proxy pooling, and retention are optional subsystems; a healthy default boot leaves them off until you decide they are useful.

## Start Server

```bash
DATA_DIR=.data spartan server
```

## The Validated Operator Loop

The canonical operator workflow is:

1. submit and verify a real job
2. promote the verified job into a template, watch, or export schedule draft
3. save the automation from its destination surface
4. use `/templates`, `/automation/watches`, and `/automation/exports` as the canonical homes for future inspection
5. reopen persisted watch checks and export outcomes from history
6. recover from failures using the saved guidance instead of starting from scratch

This workflow is validated by:

- `docs/evidence/dogfood/2026-03-20-promotion-flow-acceptance/README.md`
- `docs/evidence/dogfood/2026-03-20-saved-automation-inspection/README.md`
- `docs/evidence/dogfood/2026-03-20-populated-automation-workspace/README.md`
- `docs/evidence/dogfood/2026-03-20-failure-recovery-dogfood/README.md`

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
- AI helpers are opt-in: leave `PI_ENABLED=false` for the normal non-AI workflow, and expect `disabled` guidance rather than a degraded failure until you choose to enable it.
- Proxy pooling is opt-in: leave `PROXY_POOL_FILE` unset for the normal no-proxy path, and expect warnings only after explicitly configuring a pool file.
- Retention is optional: leave `RETENTION_ENABLED=false` until you want automated cleanup because local storage growth makes it worthwhile.
- Disabled AI/proxy/retention states are informational when you left them off by choice; follow-up actions become relevant only when you want to enable that capability.
- If the API server is already running, the CLI reads `/healthz` and the diagnostic endpoints directly.
- If the API server is offline, the CLI falls back to local checks instead of hiding recovery guidance.

## Job To Automation Workflow

### 1. Create And Verify The Source Job

A successful job is the promotion source of truth.

```bash
spartan scrape --url https://example.com --out ./out/example.json
spartan jobs list
spartan jobs get <job-id>
```

In the Web UI, the operator path starts on `/jobs/new`, then continues to `/jobs/:id` once the run succeeds.

Verify before promoting:

- the saved result is the one you want to reuse
- the target URL and runtime settings are trustworthy
- any extraction choices you want to preserve are visible on the job detail route

### 2. Promote From `/jobs/:id`

The promotion entry point is the job detail route.

Validated destinations:

- **Template** → opens a seeded draft in `/templates`
- **Watch** → opens a seeded draft in `/automation/watches`
- **Export schedule** → opens a seeded draft in `/automation/exports`

Promotion behavior that is now validated:

- the operator does not need to re-enter the verified target
- destination drafts explain what carried forward and what still needs review
- watch promotion does not silently carry screenshot-based visual diff forward from a non-browser job
- export promotion is framed as future matching job exports, not rerunning the original job

### 3. Save And Inspect From The Destination Surface

After promotion, the destination surface becomes the canonical home.

#### Templates

Use `/templates` to save and preview the promoted template.

Expected outcome:

- the saved template appears in the template library
- preview still renders the verified selectors or rules against the source page

#### Watches

Use `/automation/watches` to save the promoted watch, then run a manual check.

```bash
spartan watch check <watch-id>
spartan watch history <watch-id>
spartan watch history <watch-id> --check-id <check-id>
```

Expected outcome:

- the saved watch appears immediately in the watches workspace
- a manual check opens an immediate result summary
- `View history` reopens the same saved check through persisted history
- pagination remains correct as history grows

#### Export Schedules

Use `/automation/exports` to save the promoted export schedule, then inspect outcome history once matching exports exist.

```bash
spartan export --job-id <job-id> --schedule-id <schedule-id>
spartan export-schedule history --id <schedule-id>
spartan export --inspect-id <export-id>
spartan export --history-job-id <job-id>
```

Expected outcome:

- the saved export schedule appears immediately in the exports workspace
- export history remains inspectable from the saved schedule
- direct exports and recurring exports share one outcome-inspection model
- pagination remains correct across larger history sets

## Export Outcome Inspection

Use these commands after the save-and-inspect loop above; they are the canonical persisted-history inspection paths.

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

Use these commands after the save-and-inspect loop above; they are the canonical persisted-history inspection paths.

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

## Failure Recovery In Automation Workspaces

Failures are now part of the normal persisted operator model.

### Watch Failures

Validated failure cases:

- fetch failure, for example `http://127.0.0.1:1`, persists a failed check
- invalid selectors fail instead of silently recording empty baselines

Recovery expectations:

- the immediate result and saved history agree on the failed outcome
- failed checks do not render misleading diffs
- history keeps the recovery actions attached to the saved check
- operators can fix the selector or target, save, and rerun from `/automation/watches`

### Export Failures

Validated failure cases:

- direct export with no result file persists a failure outcome with `category=result`
- transform failures persist `category=transform` and guide the operator toward retrying without the transform or falling back to JSONL
- retryable network and timeout failures point back to `/automation/exports` with route-aware actions

Recovery expectations:

- failures are persisted in the same history model as successes
- direct and scheduled export failures stay inspectable later by export id, job history, or schedule history
- mixed success and failure pagination remains correct

Canonical inspection commands:

```bash
spartan watch history <watch-id> --check-id <check-id>
spartan export --inspect-id <export-id>
spartan export --history-job-id <job-id>
spartan export-schedule history --id <schedule-id>
```

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
