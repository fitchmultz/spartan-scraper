# Usage

Balanced 1.0 narrows Spartan Scraper to a single-node, local-first workflow:

- scrape, crawl, and research jobs
- auth vault and OAuth helpers
- templates and pipeline JS
- watches, schedules, export schedules, batches, and chains
- REST + WebSocket + MCP
- Web UI, CLI, and TUI
- local artifacts, retention, backup, and restore
- exports in `json`, `jsonl`, `csv`, `md`, and `xlsx`

Removed from this guide because they are no longer supported: GraphQL, plugins, feeds, replay tooling, multi-user/workspaces, browser extension, template A/B metrics, and cloud/database exporters.

## CLI

Binary:

```bash
./bin/spartan
```

Stable top-level commands:

- `spartan scrape`
- `spartan crawl`
- `spartan research`
- `spartan auth`
- `spartan batch`
- `spartan chains`
- `spartan watch`
- `spartan render-profiles`
- `spartan pipeline-js`
- `spartan templates`
- `spartan crawl-states`
- `spartan export`
- `spartan export-schedule`
- `spartan jobs`
- `spartan schedule`
- `spartan retention`
- `spartan backup`
- `spartan restore`
- `spartan server`
- `spartan health`
- `spartan tui`
- `spartan mcp`
- `spartan version`

### Scrape

```bash
spartan scrape --url <url> [flags]
```

Common flags:

- `--url <url>`
- `--out <path>`
- `--headless`
- `--playwright`
- `--timeout <seconds>`
- `--auth-profile <name>`
- `--auth-basic user:pass`
- `--header "Key: Value"` repeatable
- `--cookie "name=value"` repeatable

Headless login flags:

- `--login-url <url>`
- `--login-user-selector <css>`
- `--login-pass-selector <css>`
- `--login-submit-selector <css>`
- `--login-user <value>`
- `--login-pass <value>`

Examples:

```bash
spartan scrape --url https://example.com --out ./out/example.json

spartan scrape \
  --url https://example.com/dashboard \
  --headless \
  --playwright \
  --auth-profile acme \
  --out ./out/dashboard.json
```

### Crawl

```bash
spartan crawl --url <url> [flags]
```

Key flags:

- `--url <url>`
- `--max-depth <n>`
- `--max-pages <n>`
- `--out <path>`
- `--respect-robots`
- `--headless`
- `--playwright`
- `--auth-profile <name>`

Example:

```bash
spartan crawl \
  --url https://example.com \
  --max-depth 2 \
  --max-pages 200 \
  --out ./out/site.jsonl
```

### Research

```bash
spartan research --query "<text>" --urls <url1,url2,...> [flags]
```

Key flags:

- `--query "<text>"`
- `--urls <comma-separated urls>`
- `--max-depth <n>`
- `--max-pages <n>`
- `--out <path>`
- `--headless`
- `--playwright`
- `--auth-profile <name>`

Example:

```bash
spartan research \
  --query "pricing model" \
  --urls https://example.com,https://example.com/docs \
  --out ./out/research.jsonl
```

### Auth

Auth profiles live in `DATA_DIR/auth_vault.json`.

Core commands:

- `spartan auth list`
- `spartan auth set --name <profile> [auth flags]`
- `spartan auth delete --name <profile>`
- `spartan auth resolve --url <url> [--profile <name>]`
- `spartan auth vault export --out <path>`
- `spartan auth vault import --path <path>`
- `spartan auth apikey generate --name <name> [--permissions read_only|read_write]`
- `spartan auth apikey list`
- `spartan auth apikey revoke --key <key>`
- `spartan auth oauth initiate --profile <name>`
- `spartan auth oauth refresh --profile <name>`
- `spartan auth oauth revoke --profile <name>`

Profile composition flags:

- `--parent <name>` repeatable
- `--token <value>` repeatable
- `--token-kind bearer|basic|api_key`
- `--token-header <Header-Name>`
- `--token-query <param>`
- `--token-cookie <name>`
- `--preset-name <name>`
- `--preset-host <pattern>` repeatable

Examples:

```bash
spartan auth set --name acme --auth-basic user:pass --header "X-API-Key: token"
spartan auth resolve --url https://example.com --profile acme
spartan auth vault export --out ./out/auth_vault.json
```

### Templates and pipeline JS

Template commands:

- `spartan templates list`
- `spartan templates get <name>`
- `spartan templates create --name <name> --file <path>`
- `spartan templates update <name> --file <path>`
- `spartan templates delete <name>`

Pipeline JS commands:

- `spartan pipeline-js list`
- `spartan pipeline-js get <name>`
- `spartan pipeline-js create --name <name> [flags]`
- `spartan pipeline-js update <name> [flags]`
- `spartan pipeline-js delete <name>`

Render profile commands:

- `spartan render-profiles list`
- `spartan render-profiles get <name>`
- `spartan render-profiles create --name <name> --host-patterns <patterns> [flags]`
- `spartan render-profiles update <name> [flags]`
- `spartan render-profiles delete <name>`

### Jobs

- `spartan jobs list`
- `spartan jobs get <id>`
- `spartan jobs cancel <id>`
- `spartan jobs delete <id>`

### Batch jobs

- `spartan batch submit scrape --file <csv-or-json>`
- `spartan batch submit crawl --file <csv-or-json>`
- `spartan batch submit research --file <json>`
- `spartan batch status <batch-id> [--watch]`
- `spartan batch cancel <batch-id>`

### Chains

- `spartan chains list`
- `spartan chains get <chain-id>`
- `spartan chains create --file <path>`
- `spartan chains submit <chain-id>`
- `spartan chains delete <chain-id>`

### Watches

- `spartan watch add --url <url> [flags]`
- `spartan watch list`
- `spartan watch get <id>`
- `spartan watch update <id> [flags]`
- `spartan watch delete <id>`
- `spartan watch check <id>`
- `spartan watch start`

### Schedules

- `spartan schedule add --kind <scrape|crawl|research> --interval <seconds> [job flags]`
- `spartan schedule list`
- `spartan schedule delete --id <id>`

Example:

```bash
spartan schedule add --kind scrape --interval 3600 --url https://example.com
```

API note:

- `/v1/schedules` accepts `kind`, `intervalSeconds`, `specVersion`, and typed `spec`.

### Export

Supported formats:

- `json`
- `jsonl`
- `csv`
- `md`
- `xlsx`

Direct export:

```bash
spartan export --job-id <id> --format <json|jsonl|csv|md|xlsx> --out <path>
```

Examples:

```bash
spartan export --job-id 123 --format jsonl --out ./out/results.jsonl
spartan export --job-id 123 --format md --out ./out/report.md
```

### Export schedules

Supported destinations:

- `local`
- `webhook`

Commands:

- `spartan export-schedule list`
- `spartan export-schedule add [flags]`
- `spartan export-schedule get --id <id>`
- `spartan export-schedule delete --id <id>`
- `spartan export-schedule enable --id <id>`
- `spartan export-schedule disable --id <id>`
- `spartan export-schedule history --id <id>`

Example:

```bash
spartan export-schedule add \
  --name "Daily Crawl Exports" \
  --filter-kinds crawl \
  --format jsonl \
  --destination local \
  --local-path "exports/{kind}/{job_id}.jsonl"
```

### Retention, backup, and restore

Retention:

- `spartan retention status`
- `spartan retention cleanup [--dry-run]`

Backup and restore:

- `spartan backup create [-o <dir>] [--exclude-jobs]`
- `spartan backup list [--dir <dir>]`
- `spartan restore --from <archive.tar.gz> [--dry-run] [--force]`

### Service entrypoints

```bash
spartan server
spartan health
spartan tui
spartan mcp
spartan version
```

## Web UI

Run:

```bash
./bin/spartan server
make web-dev
```

Default local URL:

```text
http://localhost:5173
```

Balanced 1.0 routes:

- `/jobs`
- `/jobs/new`
- `/jobs/:id`
- `/templates`
- `/automation`
- `/settings`

The UI only exposes retained product areas. Deleted surfaces are not available behind feature flags.

## REST API

Base URL defaults to:

```text
http://127.0.0.1:8741
```

The canonical contract is [`api/openapi.yaml`](../api/openapi.yaml). Generate the web client with:

```bash
make generate
```

Important endpoint groups:

- `/healthz`
- `/v1/scrape`
- `/v1/crawl`
- `/v1/research`
- `/v1/jobs`
- `/v1/jobs/{id}`
- `/v1/jobs/{id}/results`
- `/v1/jobs/batch/*`
- `/v1/chains*`
- `/v1/watch*`
- `/v1/schedules*`
- `/v1/export-schedules*`
- `/v1/webhooks/deliveries*`
- `/v1/templates*`
- `/v1/render-profiles*`
- `/v1/pipeline-js*`
- `/v1/auth/profiles*`
- `/v1/auth/import`
- `/v1/auth/export`
- `/v1/auth/oauth/*`
- `/v1/ws`

When the server binds to a non-loopback address, API key auth is enforced automatically.

### WebSocket

`/v1/ws` provides live job and manager events.

Notes:

- browser-originated WebSocket upgrades are accepted only from loopback origins
- non-browser clients without an `Origin` header are supported
- browsers cannot set custom headers during the upgrade, so remote browser access should be fronted by a trusted deployment strategy

## MCP

Run the MCP server over stdio:

```bash
spartan mcp
```

Core tools:

- `scrape_page`
- `crawl_site`
- `research`
- `job_status`
- `job_results`
- `job_list`
- `job_cancel`
- `job_export`

Smoke example:

```bash
printf '{"id":1,"method":"tools/list"}\n' | spartan mcp
```

## Data directory

Default runtime data lives under `.data`.

Important files and directories:

- `.data/jobs.db`
- `.data/jobs/<job-id>/`
- `.data/auth_vault.json`
- `.data/render_profiles.json`
- `.data/extract_templates.json`
- `.data/pipeline_js.json`

## Storage reset policy

Balanced 1.0 is a hard storage cutover.

- New data directories are initialized automatically.
- Existing pre-cutover databases are rejected if they do not carry the Balanced 1.0 storage schema marker.
- The supported path forward is to back up the old data directory and reset to a new one.

This is deliberate: the project no longer attempts to open legacy layouts under the reduced 1.0 product boundary.

## Local CI

Required local gate:

```bash
make ci
```

Useful commands:

```bash
make install
make generate
make build
make test-ci
make ci
make ci-slow
```

`make ci-slow` provisions Playwright and runs the heavier local-fixture/browser validation lane.
