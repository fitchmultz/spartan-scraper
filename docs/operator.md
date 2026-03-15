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

If startup blocks on a legacy `.data` directory, run:

```bash
spartan reset-data
```

That archives the current data directory to `output/cutover/` and recreates `.data` for a fresh Balanced 1.0 start.

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
