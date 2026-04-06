# 5-Minute Demo

This is the fastest path from clone to visible value.

## Goal

Start the local server, open the UI, run one scrape, inspect the saved result, promote it into reusable automation, and confirm that saved watch/export history is inspectable from the destination workspaces.

This walkthrough uses the default out-of-the-box path. Leave AI, proxy pooling, and retention off; they are optional and not needed for the validated operator flow.

## Prerequisites

- Go `1.26.1`
- Node `25.9` (any `25.9.x` patch)
- pnpm `10.33.0`
- A `.tool-versions`-compatible version manager installed or the pinned versions already present on `PATH`

## Steps

```bash
git clone https://github.com/fitchmultz/spartan-scraper.git
cd spartan-scraper

make verify-toolchain
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
5. From `/jobs/:id`, use the promotion actions:
   - promote to a template draft and save it in `/templates`
   - promote to a watch draft and save it in `/automation/watches`
   - promote to an export schedule draft and save it in `/automation/exports`
6. In `/automation/watches`, run a manual check for the saved watch.
7. Open `View history` and confirm the saved check reopens through the persisted history flow.
8. In `/automation/exports`, open the saved export schedule and inspect its history after a matching export outcome exists.

Expected result:

- the dashboard shows a completed scrape job
- the result body includes `Example Domain`
- the promoted template saves and remains previewable from `/templates`
- the promoted watch saves, runs a manual check, and reopens through persisted history
- the promoted export schedule saves and exposes persisted outcome history from `/automation/exports`
- saved history remains inspectable from the destination surfaces instead of only through one-off success toasts or transient modals

## CLI Shortcut

If you want a terminal-only proof first:

```bash
./bin/spartan scrape --url https://example.com --out ./out/example.json
cat ./out/example.json
```

Expected result:

- the JSON output includes the title/content text `Example Domain`

## Failure Recovery Expectations

The validated automation workspaces persist failures too, not just successes:

- failed watch checks stay inspectable in watch history with guided next steps
- invalid watch selectors fail loudly instead of silently creating empty baselines
- failed direct exports and failed export-schedule runs persist outcome records with recovery guidance
- export history keeps pagination and mixed success/failure states intact

For the full operator recovery model, use [operator.md](operator.md).

## What You Verify

- local-first job execution
- persisted job artifacts
- promotion from a verified job into reusable automation drafts
- saved automation inspection from `/templates`, `/automation/watches`, and `/automation/exports`
- persisted watch/export history instead of fire-and-forget operator feedback
- guided recovery paths for watch/export failures

## Next Step

Use [operator.md](operator.md) for the full day-2 workflow, including watch/export inspection commands, diagnostics, and failure recovery.

## If You Want A Deeper Check

- `make ci` for the full local gate
- `docs/validation_checklist.md` for a broader validation pass
- `docs/operator.md` for the canonical operator workflow
- `docs/architecture.md` for the system structure and trade-offs
