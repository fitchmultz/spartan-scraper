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

