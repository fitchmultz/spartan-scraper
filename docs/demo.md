# 5-Minute Demo

This is the fastest path from clone to visible value.

## Goal

Start the local server, open the UI, run one scrape, and confirm that Spartan Scraper persists a real result you can inspect.

## Prerequisites

- Go `1.25.6`
- Node `24.13.0`
- pnpm `10.28.0`

## Steps

```bash
git clone <repo-url>
cd spartan-scraper

make install
make generate
make build
```

Start the backend:

```bash
./bin/spartan server
```

Expected output:

- server starts without warnings
- health endpoint is available at `http://127.0.0.1:8741/healthz`

In a second terminal, start the UI:

```bash
make web-dev
```

Open `http://localhost:5173`.

## What To Do In The UI

1. Go to the scrape form.
2. Submit `https://example.com`.
3. Wait for the job to move to `succeeded`.
4. Open the results view for that job.

Expected result:

- the dashboard shows a completed scrape job
- the result body includes `Example Domain`
- the metrics panel shows a live connection state

## CLI Shortcut

If you want a terminal-only proof first:

```bash
./bin/spartan scrape --url https://example.com --out ./out/example.json
cat ./out/example.json
```

Expected result:

- the JSON output includes the title/content text `Example Domain`

## What You Verify

- local-first job execution
- persisted artifacts
- the shared job model across UI and CLI
- the default HTTP-first fetch path

## If You Want A Deeper Check

- `make ci` for the full local gate
- `docs/validation_checklist.md` for a broader validation pass
- `docs/architecture.md` for the system structure and trade-offs
