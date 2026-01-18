# Usage

Concise, feature-complete reference for every entry point.

## CLI (spartan)

Binary: `./bin/spartan`

Global entry points:
- `spartan scrape`
- `spartan crawl`
- `spartan research`
- `spartan auth`
- `spartan export`
- `spartan schedule`
- `spartan server`
- `spartan tui`
- `spartan mcp`

### scrape

Scrape a single URL.

```
spartan scrape --url <url> [--headless] [--playwright] [--timeout <sec>] [--wait] [--wait-timeout <sec>] [--out <path>]
```

Auth options (inline or profile):
- `--auth-profile <name>`
- `--auth-basic user:pass`
- `--header "Key: Value"` (repeatable)
- `--cookie "name=value"` (repeatable)
- Login flow (headless):
  - `--login-url <url>`
  - `--login-user-selector <css>`
  - `--login-pass-selector <css>`
  - `--login-submit-selector <css>`
  - `--login-user <user>`
  - `--login-pass <pass>`

### crawl

Depth-limited, same-host crawl.

```
spartan crawl --url <url> [--max-depth <n>] [--max-pages <n>] [--headless] [--playwright]
             [--timeout <sec>] [--wait] [--wait-timeout <sec>] [--out <path>]
```

Auth flags match `scrape`.

### research

Multi-source evidence + summary (crawl or single-page per URL).

```
spartan research --query "<text>" --urls <url1,url2,...> [--max-depth <n>] [--max-pages <n>]
                [--headless] [--playwright] [--timeout <sec>] [--wait] [--wait-timeout <sec>]
                [--out <path>]
```

Auth flags match `scrape`.

### auth

Persist auth profiles (stored at `DATA_DIR/profiles.json`).

```
spartan auth list
spartan auth set --name <profile> [auth flags...]
spartan auth delete --name <profile>
```

### export

Export stored job results.

```
spartan export --job-id <id> --format <json|jsonl|md|csv> [--out <path>]
```

### schedule

Recurring jobs (stored at `DATA_DIR/schedules.json`).

```
spartan schedule add --kind <scrape|crawl|research> --interval <seconds> [job flags...]
spartan schedule list
spartan schedule delete --id <schedule-id>
```

### server

Start API + workers + scheduler:

```
spartan server
```

### tui

Terminal UI (job list + status):

```
spartan tui [--smoke]
```

`--smoke` renders a single frame and exits (CI smoke test).

### mcp

Run MCP server over stdio (JSON-RPC line protocol):

```
spartan mcp
```

Tools:
- `scrape_page`
- `crawl_site`
- `research`
- `job_status`
- `job_results`

Example:
```
printf '{"id":1,"method":"tools/list"}\n' | spartan mcp
```

## API

Base URL: `http://localhost:${PORT}` (default 8741).

Endpoints:
- `GET /healthz`
- `POST /v1/scrape`
- `POST /v1/crawl`
- `POST /v1/research`
- `GET /v1/jobs`
- `GET /v1/jobs/{id}`
- `GET /v1/jobs/{id}/results`

OpenAPI: `api/openapi.yaml`

### Example (scrape)

```
curl -sS -X POST "http://localhost:8741/v1/scrape" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","headless":false,"playwright":false,"timeoutSeconds":30}'
```

## Web UI

Dev server:
```
make web-dev
```

The UI connects to the API server (same `PORT`).

Preview the production build (after `make build`):
```
cd web && pnpm exec vite preview --host 127.0.0.1 --port 4173
```

## Scripts

### Stress test

```
scripts/stress_test.sh --help
```

Key options:
- `--openai-docs`
- `--headless` / `--use-playwright`
- `--targets <csv>`
- `--wait-timeout <sec>`

Outputs: `out/stress/`

## Configuration

`.env` / `.env.example`:
- `PORT`
- `DATA_DIR`
- `USER_AGENT`
- `MAX_CONCURRENCY`
- `REQUEST_TIMEOUT_SECONDS`
- `RATE_LIMIT_QPS`
- `RATE_LIMIT_BURST`
- `MAX_RETRIES`
- `RETRY_BASE_MS`
- `USE_PLAYWRIGHT`

## Outputs

Jobs stored under `DATA_DIR/jobs/<id>/results.jsonl`.
- Scrape: single JSON object.
- Crawl: JSONL, one page per line.
- Research: single JSON object (summary + evidence).

## CI Coverage

`make ci` runs unit + end-to-end coverage across:
- CLI (all subcommands, help, auth profiles, export)
- API (scrape/crawl/research/jobs/results)
- MCP (tools list + scrape_page)
- Scheduler (job creation via interval)
- Web (TypeScript build + preview smoke test)
- External auth targets (public demo sites + httpbin basic auth)
