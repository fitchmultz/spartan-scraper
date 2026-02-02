# Usage

Concise, feature-complete reference for every entry point.

## CLI (spartan)

Binary: `./bin/spartan`

Global entry points:
- `spartan scrape`
- `spartan crawl`
- `spartan research`
- `spartan auth`
- `spartan render-profiles`
- `spartan pipeline-js`
- `spartan templates`
- `spartan crawl-states`
- `spartan export`
- `spartan jobs`
- `spartan schedule`
- `spartan server`
- `spartan health`
- `spartan tui`
- `spartan mcp`

### scrape

Scrape a single URL.

```
spartan scrape --url <url> [--headless] [--playwright] [--timeout <sec>] [--wait] [--wait-timeout <sec>] [--out <path>]
```

Auth options (inline, profile, or preset via `spartan auth resolve`):
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

### render-profiles

List configured render profiles.

  - `spartan render-profiles list`

### pipeline-js

List configured pipeline JavaScript scripts.

  - `spartan pipeline-js list`

### plugin

Manage third-party WASM plugins (stored at `DATA_DIR/plugins/`).

  - `spartan plugin list` - List all installed plugins
  - `spartan plugin install --path <dir>` - Install a plugin from a directory
  - `spartan plugin uninstall --name <name>` - Remove an installed plugin
  - `spartan plugin enable --name <name>` - Enable a plugin
  - `spartan plugin disable --name <name>` - Disable a plugin
  - `spartan plugin configure --name <name> --key <key> --value <value>` - Set a plugin configuration value
  - `spartan plugin info --name <name>` - Show detailed plugin information

Plugin directory structure:
```
my-plugin/
  manifest.json     # Plugin metadata (name, version, hooks, permissions)
  plugin.wasm       # Compiled WASM binary
  config.json       # Optional: default configuration
```

Manifest example:
```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "My custom plugin",
  "author": "Developer Name",
  "hooks": ["pre_fetch", "post_extract"],
  "permissions": ["network"],
  "wasm_path": "plugin.wasm",
  "enabled": true,
  "priority": 10
}
```

Supported hooks: `pre_fetch`, `post_fetch`, `pre_extract`, `post_extract`, `pre_output`, `post_output`

Supported permissions: `network`, `filesystem`, `env`

### auth

Persist auth profiles (stored at `DATA_DIR/auth_vault.json`).

  - `spartan auth list`
  - `spartan auth set --name <profile> [auth flags...]`
  - `spartan auth delete --name <profile>`
  - `spartan auth resolve --url <url> [--profile <name>]`
  - `spartan auth vault export --out <path>`
  - `spartan auth vault import --path <path>`

Auth flags:
- `--parent <name>` (repeatable)
- `--auth-basic user:pass`
- `--token <value>` (repeatable)
- `--token-kind bearer|basic|api_key`
- `--token-header <Header-Name>`
- `--token-query <param>`
- `--token-cookie <name>`
- `--header "Key: Value"` (repeatable)
- `--cookie "name=value"` (repeatable)
- Login flow (headless):
  - `--login-url <url>`
  - `--login-user-selector <css>`
  - `--login-pass-selector <css>`
  - `--login-submit-selector <css>`
  - `--login-user <user>`
  - `--login-pass <pass>`

Presets:
- `--preset-name <name>` + `--preset-host <pattern>` (repeatable) to map host patterns to a profile.

### templates

List and manage extraction templates.

  - `spartan templates list`

Spartan supports structured extraction using templates. Templates define CSS selectors, JSON-LD extraction, Regex rules, and schema validation.

**Built-in templates:**
- `default`: Title, description, H1, meta tags.
- `article`: Title, author, date, content, JSON-LD Article.
- `product`: Name, price, currency, JSON-LD Product.

**Usage:**
```bash
spartan scrape --url ... --extract-template product
spartan scrape --url ... --extract-validate
```

**Custom Templates:**
Create `DATA_DIR/extract_templates.json`:
```json
{
  "templates": [
    {
      "name": "custom-blog",
      "selectors": [
        {"name": "title", "selector": "h1.entry-title", "attr": "text", "trim": true},
        {"name": "author", "selector": ".author-name", "attr": "text"}
      ],
      "normalize": {
        "titleField": "title"
      }
    }
  ]
}
```

### AI-Powered Extraction

Extract structured data from HTML using LLM (Large Language Model) providers. This feature enables natural language extraction without writing CSS selectors.

**Supported Providers:**
- **OpenAI** (GPT-4o-mini, GPT-4, etc.)
- **Anthropic** (Claude 3 Haiku, Sonnet, etc.)
- **Ollama** (local LLMs like Llama 3.1)

**Configuration** (`.env`):
```
AI_PROVIDER=openai              # openai, anthropic, or ollama
AI_API_KEY=sk-...               # API key for cloud providers
AI_MODEL=gpt-4o-mini            # Optional: defaults per provider
AI_TIMEOUT_SECONDS=60           # 5-300 seconds
AI_MAX_TOKENS=4096
AI_TEMPERATURE=0.1              # 0.0-1.0 (lower = more consistent)
OLLAMA_URL=http://localhost:11434  # For local Ollama
```

**CLI Usage:**
```bash
# Natural language extraction
spartan scrape --url https://example.com --ai-extract --ai-prompt "extract all product names and prices"

# Schema-guided extraction with specific fields
spartan crawl --url https://example.com --ai-extract --ai-mode schema_guided --ai-fields "title,price,rating"
```

**API Usage:**
```bash
# Preview extraction without creating a job
curl -sS -X POST "http://localhost:8741/v1/extract/ai-preview" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "html": "<html>...</html>",
    "mode": "natural_language",
    "prompt": "Extract all product names and prices",
    "fields": ["name", "price"]
  }'
```

**Extraction Modes:**
- `natural_language`: Describe what to extract in plain English
- `schema_guided`: Provide example field names to guide extraction

**Features:**
- Automatic HTML cleaning (removes scripts, styles, comments)
- Content-based caching (24h TTL) to reduce API costs
- Graceful fallback if AI extraction fails
- Token usage tracking
- Confidence scores per extraction

### crawl-states

List incremental crawl states (ETags/Last-Modified tracking).

```
spartan crawl-states list [--limit <n>]
```

### Render Profiles

List configured render profiles:
- `spartan render-profiles list`

To customize rendering behavior per site (e.g., forcing headless, increasing timeouts, blocking resources), create a `render_profiles.json` in your `DATA_DIR` (default `.data`).

**Schema example (`.data/render_profiles.json`):**

```json
{
  "profiles": [
    {
      "name": "complex-spa",
      "hostPatterns": ["*.example-spa.com", "app.example.com"],
      "forceEngine": "chromedp",
      "wait": {
        "mode": "network_idle",
        "networkIdleQuietMs": 500
      },
      "block": {
        "resourceTypes": ["image", "font", "media"]
      }
    },
    {
      "name": "slow-loader",
      "hostPatterns": ["slow.com"],
      "timeouts": {
        "maxRenderMs": 60000
      },
      "preferHeadless": true
    }
  ]
}
```

### Pipeline hooks

Optional pipeline flags (repeatable):
- `--pre-processor <name>`
- `--post-processor <name>`
- `--transformer <name>`

These map to the standardized plugin interface in `internal/pipeline`.

### Data Transformation

Spartan supports transforming extracted data using **JMESPath** and **JSONata** query languages. These are useful for reshaping, filtering, and aggregating scraped data without writing custom code.

**When to use each:**

- **JMESPath**: Simpler syntax, great for projections, filtering, and basic transformations. Use when you need to extract nested fields or filter arrays.
- **JSONata**: More powerful, supports complex aggregations, calculations, and custom functions. Use when you need to compute values, group data, or perform conditional logic.

**JMESPath Examples:**

```bash
# Extract just the titles from a crawl
spartan scrape --url https://example.com --transformer 'jmespath:{title: title, url: url}'

# Filter to only items with a specific field
spartan crawl --url https://example.com --transformer 'jmespath:items[?status == `active`]'
```

Common JMESPath patterns:
- Projection: `{name: name, price: price}` - Select specific fields
- Filtering: `items[?price > 100]` - Filter arrays by condition
- Slicing: `items[0:10]` - Get first 10 items
- Sorting: `sort_by(items, &price)` - Sort by field
- Counting: `length(items)` - Count array items

**JSONata Examples:**

```bash
# Calculate total price with tax
spartan scrape --url https://example.com --transformer 'jsonata:{"item": name, "total": price * 1.08}'

# Filter and transform in one expression
spartan crawl --url https://example.com --transformer 'jsonata:items[price > 100].{"name": name, "cost": price}'
```

Common JSONata patterns:
- Projection with computed fields: `{"name": name, "total": price * quantity}`
- Conditional logic: `{"status": price > 100 ? "premium" : "standard"}`
- Aggregation: `$sum(items.price)` - Sum all prices
- Grouping: `items{category: $count($)}` - Count by category
- Array mapping: `items.{"title": title, "url": link}`

**API Usage:**

Validate an expression before using:
```bash
curl -sS -X POST "http://localhost:8741/v1/transform/validate" \
  -H "Content-Type: application/json" \
  -d '{"expression":"{title: title, url: url}","language":"jmespath"}'
```

Preview transformation on job results:
```bash
curl -sS -X POST "http://localhost:8741/v1/jobs/abc123/preview-transform" \
  -H "Content-Type: application/json" \
  -d '{"expression":"items[?price > 100]","language":"jmespath","limit":5}'
```

**Web UI:**

In the Results Explorer, switch to the "Transform" tab to:
1. Enter a JMESPath or JSONata expression
2. See real-time validation feedback
3. Preview transformed results before exporting

### JS per-target scripts (headless)

List configured pipeline JavaScript scripts:
- `spartan pipeline-js list`

Place `pipeline_js.json` in your `DATA_DIR` (default `.data`):

```json
{
  "scripts": [
    {
      "name": "spa-boost",
      "hostPatterns": ["*.example.com"],
      "engine": "chromedp",
      "preNav": "window.localStorage.setItem('exp','1')",
      "postNav": "document.body.click()",
      "selectors": ["#root"]
    }
  ]
}
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

### jobs

Manage background jobs.

```
spartan jobs list [--limit <n>] [--offset <n>] [--status <queued|running|succeeded|failed|canceled>]
spartan jobs get --id <id>
spartan jobs cancel --id <id>
```

### server

Start API + workers + scheduler:

```
spartan server
```

### health

Check system health (database connection, etc).

```
spartan health
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
- `job_list`
- `job_cancel`
- `job_export`

Example:
```
printf '{"id":1,"method":"tools/list"}\n' | spartan mcp
```

## API

Base URL: `http://${BIND_ADDR}:${PORT}` (defaults to `http://127.0.0.1:8741`).

Note: if you set `BIND_ADDR=0.0.0.0` (bind all interfaces), clients should connect via
`http://127.0.0.1:${PORT}` locally or via the machine's LAN IP/hostname from other devices.

Endpoints:
- `GET /healthz`
- `GET /v1/auth/profiles`
- `PUT /v1/auth/profiles/{name}`
- `DELETE /v1/auth/profiles/{name}`
- `POST /v1/auth/import`
- `POST /v1/auth/export`
- `POST /v1/scrape`
- `POST /v1/crawl`
- `POST /v1/research`
- `GET /v1/jobs`
- `GET /v1/jobs/{id}`
- `DELETE /v1/jobs/{id}`
- `GET /v1/jobs/{id}/results`
- `GET /v1/schedules`
- `POST /v1/schedules`
- `DELETE /v1/schedules/{id}`
- `GET /v1/templates`
- `GET /v1/crawl-states`
- `POST /v1/extract/ai-preview`
- `POST /v1/extract/ai-template-generate`

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

### Export Schedules

The Web UI provides a complete interface for managing automated export schedules. Export schedules automatically export job results when jobs complete matching specified filter criteria.

**Features:**
- **List View**: See all configured export schedules with status, filters, destination, and format
- **Create/Edit**: Configure new export schedules with name, filters, destination, and retry settings
- **Enable/Disable**: Toggle schedules on/off directly from the list
- **Delete**: Remove schedules with confirmation
- **History**: View export execution history per schedule with status, timestamps, and stats

**Filter Criteria:**
- Job kinds (scrape, crawl, research)
- Job status (completed, succeeded, failed, canceled)
- Tags (all must match)
- Has results (only jobs with non-empty results)

**Destination Types:**
- **Local File**: Save exports to local filesystem path
- **Webhook**: POST export data to a webhook URL
- **Cloud Storage**: S3, GCS, or Azure Blob Storage (credentials via environment/IAM)

**Path Templates:**
Use variables in path templates: `{job_id}`, `{timestamp}`, `{kind}`, `{format}`

Example: `exports/{kind}/{job_id}-{timestamp}.{format}`

**CLI Parity:**
The Web UI mirrors the functionality of:
```
spartan manage export-schedule list
spartan manage export-schedule create --name "..." ...
spartan manage export-schedule update --id "..." ...
spartan manage export-schedule delete --id "..."
spartan manage export-schedule history --id "..."
```

## Feed Monitoring

Monitor RSS/Atom feeds and automatically create scrape jobs for new items.

### CLI

- `spartan feed add --url <url> [--type rss|atom|auto] [--interval <seconds>] [--auto-scrape]` - Add a feed
- `spartan feed list [--all]` - List all feeds
- `spartan feed get --id <id>` - Get feed details
- `spartan feed update --id <id> [options]` - Update a feed
- `spartan feed delete --id <id>` - Delete a feed
- `spartan feed check --id <id>` - Manually check a feed
- `spartan feed items --id <id>` - List seen items for a feed
- `spartan feed start` - Start the feed scheduler daemon

### Web UI

The Feed Manager section provides a visual interface for:

1. **Viewing feeds** - See all configured feeds with status, last check time, and error state
2. **Adding feeds** - Enter URL, select type (auto/rss/atom), set check interval
3. **Editing feeds** - Modify feed settings (URL, interval, auto-scrape)
4. **Checking feeds** - Manually trigger a check and see new items found
5. **Viewing items** - Browse all seen items for a feed with links

Feed settings:
- **URL**: RSS/Atom feed URL
- **Type**: Auto-detect, RSS 2.0, or Atom 1.0
- **Check interval**: How often to poll (minimum 60 seconds)
- **Enabled**: Whether the feed is actively checked
- **Auto-scrape**: Automatically create scrape jobs for new items

## Webhook Delivery Debugging

Debug and monitor webhook delivery history to troubleshoot integration issues.

### API

- `GET /v1/webhooks/deliveries` - List webhook deliveries with optional job_id filter and pagination (limit/offset)
- `GET /v1/webhooks/deliveries/{id}` - Get detailed information about a specific delivery

### Web UI

The Webhook Deliveries section provides a visual dashboard for:

1. **Viewing deliveries** - See all webhook deliveries with status, timestamps, and metadata
2. **Filtering** - Filter by job ID or status (pending, delivered, failed)
3. **Pagination** - Navigate through delivery history with server-side pagination
4. **Detail view** - Drill down into individual deliveries to see full request/response details

Delivery statuses:
- **Pending** - Webhook is queued for delivery
- **Delivered** - Webhook was successfully delivered (2xx response)
- **Failed** - Webhook delivery failed after all retry attempts

The detail view shows:
- Delivery metadata (ID, event type, job ID, URL)
- Status and number of attempts
- Response code (if available)
- Timestamps (created, updated, delivered)
- Error message (if failed)
- Full technical details (JSON)

### Common Error Patterns

**Connection timeouts**: Check that the webhook URL is accessible from the server and firewalls allow outbound connections.

**4xx errors**: Verify the webhook endpoint is configured correctly. Common issues include wrong URL path, missing authentication headers, or incorrect payload format.

**5xx errors**: The receiving server is experiencing issues. Check the receiving service's logs and health status.

**SSL/TLS errors**: Ensure the webhook URL uses valid SSL certificates. For self-signed certificates, you may need to configure the receiving server appropriately.

### CLI Parity

The Web UI mirrors the functionality of:
```
GET /v1/webhooks/deliveries?job_id=<id>&limit=<n>&offset=<n>
GET /v1/webhooks/deliveries/<id>
```

## Data Retention

Manage data retention policies to control disk usage and prevent storage exhaustion.

### CLI

```bash
# View retention status
spartan retention status

# Preview what would be cleaned (dry-run)
spartan retention cleanup --dry-run

# Run cleanup immediately
spartan retention cleanup

# Cleanup with overrides
spartan retention cleanup --older-than=7 --kind=scrape
spartan retention cleanup --force  # Run even if retention disabled
```

### Environment Variables

- `RETENTION_ENABLED` - Enable automatic retention (default: false)
- `RETENTION_JOB_DAYS` - Max age for jobs in days (default: 30, 0 = unlimited)
- `RETENTION_CRAWL_STATE_DAYS` - Max age for crawl states in days (default: 90, 0 = unlimited)
- `RETENTION_MAX_JOBS` - Max total jobs to keep (default: 10000, 0 = unlimited)
- `RETENTION_MAX_STORAGE_GB` - Max storage in GB (default: 10, 0 = unlimited)
- `RETENTION_CLEANUP_INTERVAL_HOURS` - Hours between cleanup runs (default: 24)
- `RETENTION_DRY_RUN_DEFAULT` - Default dry-run mode (default: false)

### Web UI

The Data Retention panel (below Webhook Deliveries) provides:

- **Status Overview**: Current storage usage, total jobs, and eligible cleanup count
- **Configuration Display**: Current retention policy settings
- **Cleanup Controls**:
  - Dry-run toggle for safe preview
  - Optional job kind filter (scrape/crawl/research)
  - Optional age override
  - Confirmation dialog for destructive operations
- **Results Display**: Summary of cleanup operations with space reclaimed

### Safety Features

- Dry-run is the default mode - no data is deleted without explicit confirmation
- Cleanup shows a confirmation dialog when not in dry-run mode
- Failed artifact deletions preserve database records (no orphaned metadata)
- Jobs are prioritized for cleanup: failed first, then succeeded, then others

### API

```
GET /v1/retention/status
POST /v1/retention/cleanup
```

Example:
```bash
# Get status
curl http://localhost:8741/v1/retention/status

# Dry-run cleanup
curl -X POST http://localhost:8741/v1/retention/cleanup \
  -H "Content-Type: application/json" \
  -d '{"dryRun": true}'

# Actual cleanup with filter
curl -X POST http://localhost:8741/v1/retention/cleanup \
  -H "Content-Type: application/json" \
  -d '{"dryRun": false, "kind": "scrape", "olderThan": 7}'
```

## Content Deduplication

The Deduplication Explorer provides a visual interface for analyzing content fingerprints and finding duplicate pages across jobs. This helps improve crawl efficiency and prevent wasted storage.

### How Deduplication Works

Spartan uses [simhash](https://en.wikipedia.org/wiki/SimHash) fingerprints to detect similar content:
- Each page's content is hashed into a 64-bit simhash value
- Similar pages will have simhash values with small Hamming distances
- The system tracks content across all jobs to detect duplicates

**Distance Thresholds:**
- **0 bits**: Exact match (identical content)
- **1-3 bits**: Near duplicate (very similar content)
- **4-8 bits**: Similar (related content)
- **9-16 bits**: Distinct (different content)

### Web UI

The Deduplication panel provides three tabs:

1. **Find Duplicates** - Search for similar content by simhash:
   - Enter a simhash value to find matching content
   - Adjust the threshold slider to control similarity (0-16)
   - Results show matching URLs with their Hamming distance

2. **URL History** - View all indexed versions of a URL:
   - Enter a URL to see its content history across jobs
   - Shows simhash values and when each version was indexed
   - Useful for tracking content changes over time

3. **Statistics** - Overview of deduplication data:
   - Total content fingerprints indexed
   - Number of unique URLs
   - Number of jobs with indexed content
   - Count of duplicate URLs across jobs

### API

```
GET /v1/dedup/duplicates?simhash={simhash}&threshold={threshold}
GET /v1/dedup/history?url={url}
GET /v1/dedup/stats
DELETE /v1/dedup/job/{jobId}
```

Example:
```bash
# Find duplicates for a simhash
curl "http://localhost:8741/v1/dedup/duplicates?simhash=1234567890&threshold=3"

# Get content history for a URL
curl "http://localhost:8741/v1/dedup/history?url=https://example.com/page"

# Get deduplication statistics
curl "http://localhost:8741/v1/dedup/stats"

# Clean up dedup entries for a deleted job
curl -X DELETE "http://localhost:8741/v1/dedup/job/abc123"
```

### CLI

Deduplication data is automatically managed during job execution. To clean up entries for deleted jobs:

```bash
# Remove dedup entries for a specific job
spartan jobs delete --id <job-id>
# This automatically cleans up associated dedup entries
```

### Use Cases

- **Pre-crawl checking**: Query if a URL has been indexed recently to avoid redundant crawling
- **Content analysis**: Identify near-duplicate pages on your site for consolidation
- **Change detection**: Track when content at a URL has significantly changed
- **Storage optimization**: Identify duplicate content across jobs for cleanup

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

## Prerequisites

This project uses a polyglot toolchain. The following are required:

- **Go** 1.25.6 (CLI + API + TUI)
- **Node.js** 24.13.0 (web frontend + OpenAPI generation)
- **pnpm** 10.28.0 (Node package manager)

Tool versions are pinned in `.tool-versions`. Use `asdf` or `mise` to install.

No additional tools (ripgrep, perl, etc.) are required for `make generate`.

## Configuration

`.env` / `.env.example`:
- `PORT`
- `BIND_ADDR` (default `127.0.0.1`; set `0.0.0.0` to expose beyond localhost)
- `SERVER_READ_HEADER_TIMEOUT_SECONDS` (default `10`)
- `SERVER_READ_TIMEOUT_SECONDS` (default `30`)
- `SERVER_WRITE_TIMEOUT_SECONDS` (default `60`)
- `SERVER_IDLE_TIMEOUT_SECONDS` (default `120`)
- `DATA_DIR`
- `USER_AGENT`
- `LOG_LEVEL` (default `info`)
- `LOG_FORMAT` (default `text`)
- `MAX_CONCURRENCY`
- `REQUEST_TIMEOUT_SECONDS`
- `RATE_LIMIT_QPS`
- `RATE_LIMIT_BURST`
- `MAX_RETRIES`
- `RETRY_BASE_MS`
- `MAX_RESPONSE_BYTES` (default `10485760`)
- `USE_PLAYWRIGHT`
- AI extraction (optional):
  - `AI_PROVIDER` (`openai`, `anthropic`, or `ollama`)
  - `AI_API_KEY`
  - `AI_MODEL`
  - `AI_TIMEOUT_SECONDS` (default `60`)
  - `AI_MAX_TOKENS` (default `4096`)
  - `AI_TEMPERATURE` (default `0.1`)
  - `OLLAMA_URL` (default `http://localhost:11434`)
- Auth overrides:
  - `AUTH_BASIC`
  - `AUTH_BEARER`
  - `AUTH_API_KEY`
  - `AUTH_API_KEY_HEADER`
  - `AUTH_API_KEY_QUERY`
  - `AUTH_API_KEY_COOKIE`
  - `AUTH_HEADER_*`
  - `AUTH_COOKIE_*`

## Outputs

Jobs stored under `DATA_DIR/jobs/<id>/results.jsonl`.
- Scrape: single JSON object.
- Crawl: JSONL, one page per line.
- Research: single JSON object (summary + evidence + simhash dedup + clusters + citations + confidence).

## CI Coverage

`make ci` (Fast/Local) runs unit + integration coverage across:
- CLI (all subcommands, help, auth profiles, export)
- API (scrape/crawl/research/jobs/results)
- MCP (tools list + scrape_page)
- Scheduler (job creation via interval)
- Web (TypeScript build + unit tests)

`make ci-slow` (Stress/Network) executes `scripts/stress_test.sh` and E2E tests (`internal/e2e`), which validate:
- Real-world targets (no mocks)
- Full end-to-end workflows (CLI → API → Worker → Exporter)
- External auth targets and headless behaviors
- Web preview smoke test

