# Operator Guide

## Deployment Model

- Supported deployment model: single node, trusted operator, local disk, SQLite.
- Default data root: `.data`.
- Job artifacts live under `.data/jobs/<job-id>/`.
- Each completed or failed job writes `.data/jobs/<job-id>/manifest.json`.

## Runtime Expectations

- Non-loopback API binds require API-key protection.
- Browser engines are local process dependencies; keep Chromium/Playwright availability aligned with your workload.
- Backup and restore must include the SQLite database and the full `.data/jobs/` tree.

## Storage Reset

- If startup reports a legacy storage schema, stop the server and archive the old data directory.
- Start with a fresh data directory for the hard-cutover build.
