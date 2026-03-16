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
- `spartan ai`
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
- `spartan reset-data`
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
- `--proxy <url>`
- `--proxy-username <value>`
- `--proxy-password <value>`
- `--proxy-region <value>`
- `--proxy-tag <value>` repeatable
- `--exclude-proxy-id <value>` repeatable
- `--ai-extract`
- `--ai-mode natural_language|schema_guided`
- `--ai-prompt "<instructions>"` for natural-language mode
- `--ai-schema '{"field":"example"}'` for schema-guided mode
- `--ai-fields "field1,field2"`

Direct `--proxy ...` overrides and proxy-pool selection hints are mutually exclusive.

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

spartan scrape \
  --url https://example.com/product \
  --ai-extract \
  --ai-mode schema_guided \
  --ai-schema '{"title":"Example","price":"$19.99"}' \
  --ai-fields "title,price"
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
- `--proxy <url>`
- `--proxy-username <value>`
- `--proxy-password <value>`
- `--proxy-region <value>`
- `--proxy-tag <value>` repeatable
- `--exclude-proxy-id <value>` repeatable
- `--ai-extract`
- `--ai-mode natural_language|schema_guided`
- `--ai-prompt "<instructions>"` for natural-language mode
- `--ai-schema '{"field":"example"}'` for schema-guided mode
- `--ai-fields "field1,field2"`

Example:

```bash
spartan crawl \
  --url https://example.com \
  --max-depth 2 \
  --max-pages 200 \
  --out ./out/site.jsonl

spartan crawl \
  --url https://example.com/catalog \
  --max-depth 2 \
  --ai-extract \
  --ai-prompt "Extract the title, price, and availability from each crawled page" \
  --ai-fields "title,price,availability"
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
- `--proxy <url>`
- `--proxy-username <value>`
- `--proxy-password <value>`
- `--proxy-region <value>`
- `--proxy-tag <value>` repeatable
- `--exclude-proxy-id <value>` repeatable
- `--ai-extract`
- `--ai-mode natural_language|schema_guided`
- `--ai-prompt "<instructions>"` for natural-language mode
- `--ai-schema '{"field":"example"}'` for schema-guided mode
- `--ai-fields "field1,field2"`
- `--agentic`
- `--agentic-instructions "<instructions>"`
- `--agentic-max-rounds <1-3>`
- `--agentic-max-follow-up-urls <1-10>`

Examples:

```bash
spartan research \
  --query "pricing model" \
  --urls https://example.com,https://example.com/docs \
  --out ./out/research.jsonl

spartan research \
  --query "pricing model" \
  --urls https://example.com/pricing,https://example.com/support \
  --ai-extract \
  --ai-prompt "Extract the pricing model, contract terms, and support commitments from each source" \
  --ai-fields "pricing_model,contract_terms,support_commitments"

spartan research \
  --query "pricing model" \
  --urls https://example.com,https://example.com/docs \
  --agentic \
  --agentic-instructions "Prioritize pricing, contract terms, and support commitments" \
  --agentic-max-rounds 2 \
  --agentic-max-follow-up-urls 4
```

### AI authoring

```bash
spartan ai preview [flags]
spartan ai template [flags]
spartan ai template-debug [flags]
spartan ai render-profile [flags]
spartan ai render-profile-debug [flags]
spartan ai pipeline-js [flags]
spartan ai pipeline-js-debug [flags]
spartan ai research-refine [flags]
spartan ai export-shape [flags]
```

These commands run the same bounded AI authoring workflows as the REST and Web surfaces, but without creating jobs. Preview/template/render-profile/pipeline authoring commands also accept repeatable `--image-file <path>` flags for request-scoped reference images. Those attachments are bounded visual context only and are not persisted as job artifacts.

Preview flags:

- `--url <url>` or `--html-file <path>` / `--html '<html>...'`
- `--mode natural_language|schema_guided`
- `--prompt "<instructions>"` for natural-language mode
- `--schema '{"field":"example"}'` for schema-guided mode
- `--fields "field1,field2"`
- `--image-file <path>` repeatable request-scoped reference images
- `--headless`
- `--playwright`
- `--visual` to capture a screenshot and include multimodal visual context when fetching a URL
- `--out <path>`

Template-generation flags:

- `--url <url>` or `--html-file <path>` / `--html '<html>...'`
- `--description "<what to extract>"`
- `--sample-fields "field1,field2"`
- `--image-file <path>` repeatable request-scoped reference images
- `--headless`
- `--playwright`
- `--visual` to capture a screenshot and include multimodal visual context when fetching a URL
- `--out <path>`

Template-debug flags:

- `--url <url>` or `--html-file <path>` / `--html '<html>...'`
- `--template-name <saved-template>` or `--template-file <path>`
- `--instructions "<repair guidance>"`
- `--image-file <path>` repeatable request-scoped reference images
- `--headless`
- `--playwright`
- `--visual` to capture a screenshot and include multimodal visual context when fetching a URL
- `--out <path>`

Render-profile flags:

- `--url <url>`
- `--name <profile-name>`
- `--host-patterns "example.com,*.example.com"`
- `--instructions "<fetch-behavior guidance>"`
- `--image-file <path>` repeatable request-scoped reference images
- `--headless`
- `--playwright`
- `--visual` to capture a screenshot and include multimodal visual context when fetching a URL
- `--out <path>`

Render-profile-debug flags:

- `--url <url>`
- `--profile-name <saved-profile>` or `--profile-file <path>`
- `--instructions "<tuning guidance>"`
- `--image-file <path>` repeatable request-scoped reference images
- `--headless`
- `--playwright`
- `--visual` to capture a screenshot and include multimodal visual context for tuning
- `--out <path>`

Pipeline-JS flags:

- `--url <url>`
- `--name <script-name>`
- `--host-patterns "example.com,*.example.com"`
- `--instructions "<browser-automation guidance>"`
- `--image-file <path>` repeatable request-scoped reference images
- `--headless`
- `--playwright`
- `--visual` to capture a screenshot and include multimodal visual context when fetching a URL
- `--out <path>`

Pipeline-JS-debug flags:

- `--url <url>`
- `--script-name <saved-script>` or `--script-file <path>`
- `--instructions "<tuning guidance>"`
- `--image-file <path>` repeatable request-scoped reference images
- `--headless`
- `--playwright`
- `--visual` to capture a screenshot and include multimodal visual context for tuning
- `--out <path>`

Research-refine flags:

- `--job-id <research-job-id>` or `--result-file <path>`
- `--instructions "<rewrite guidance>"`
- `--out <path>`

`--result-file` accepts a single research result as a JSON object, a single-item JSON array, or a single-result JSONL file.

Export-shape flags:

- `--job-id <job-id>` or `--result-file <path>`
- `--kind <scrape|crawl|research>` when `--result-file` needs an explicit kind hint
- `--format <md|csv|xlsx>`
- `--schedule-id <id>` to seed from an existing export schedule shape
- `--shape-file <path>` to seed from a saved shape JSON file
- `--instructions "<field-selection guidance>"`
- `--out <path>`

`--result-file` accepts a representative scrape/crawl/research result artifact from disk, and `--shape-file` accepts a JSON `ExportShapeConfig` object.

Transform flags:

- `--job-id <job-id>` or `--result-file <path>`
- `--schedule-id <id>` to seed from an existing export schedule transform
- `--transform-file <path>` to seed from a saved transform JSON file
- `--language <jmespath|jsonata>` optional preferred language for the generated transform
- `--expression "<current transform expression>"` optional current transform to tune
- `--instructions "<projection/filter guidance>"`
- `--out <path>`

`--result-file` accepts a representative saved result artifact from disk, and `--transform-file` accepts a JSON `ResultTransformConfig` object. The generated transform is validated against bounded sample records before it is returned.

Examples:

```bash
spartan ai preview \
  --url https://example.com/product \
  --prompt "Extract the title, price, and availability"

spartan ai preview \
  --html-file ./fixtures/product.html \
  --mode schema_guided \
  --schema '{"title":"Example","price":"$19.99"}'

spartan ai template \
  --url https://example.com/product \
  --description "Extract the title and price" \
  --image-file ./fixtures/product-hero.png \
  --image-file ./fixtures/product-gallery.png

spartan ai template \
  --url https://example.com/product \
  --description "Extract the product title, price, and availability" \
  --out ./out/product-template.json

spartan ai template \
  --html-file ./fixtures/product.html \
  --description "Extract the pricing table and support commitments"

spartan ai template-debug \
  --url https://example.com/product \
  --template-name product \
  --instructions "Prefer visible headings and avoid brittle nth-child selectors" \
  --visual

spartan ai render-profile \
  --url https://example.com/app \
  --instructions "Wait for the dashboard shell and prefer headless mode when the HTTP shell is sparse" \
  --visual

spartan ai render-profile-debug \
  --url https://example.com/app \
  --profile-name example-app \
  --instructions "Prefer a stable selector wait for the visible dashboard shell" \
  --visual

spartan ai pipeline-js \
  --url https://example.com/app \
  --instructions "Wait for the main dashboard shell and reset scroll position before extraction" \
  --visual

spartan ai pipeline-js-debug \
  --url https://example.com/app \
  --script-name example-app \
  --instructions "Prefer selector waits over post-nav JS where possible" \
  --visual

spartan ai research-refine \
  --job-id <research-job-id> \
  --instructions "Condense this into an operator-ready brief with the strongest evidence first"

spartan ai export-shape \
  --job-id <job-id> \
  --format md \
  --instructions "Prioritize operator-facing summary and pricing fields"

spartan ai export-shape \
  --schedule-id <export-schedule-id> \
  --job-id <job-id> \
  --format csv \
  --out ./out/export-shape.json

spartan ai transform \
  --job-id <job-id> \
  --language jmespath \
  --instructions "Project the URL, title, and pricing fields for export"

spartan ai transform \
  --schedule-id <export-schedule-id> \
  --job-id <job-id>

spartan ai transform \
  --result-file ./out/crawl.jsonl \
  --transform-file ./out/current-transform.json \
  --language jsonata \
  --expression '$.{"url": url}' \
  --out ./out/transform.json
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

Batch submit commands for scrape, crawl, and research accept the same AI extraction flags as their single-job counterparts:

- `--ai-extract`
- `--ai-mode natural_language|schema_guided`
- `--ai-prompt "<instructions>"`
- `--ai-schema '{"field":"example"}'`
- `--ai-fields "field1,field2"`

Batch research also accepts bounded agentic controls:

- `--agentic`
- `--agentic-instructions "<instructions>"`
- `--agentic-max-rounds <1-3>`
- `--agentic-max-follow-up-urls <1-10>`

### Chains

- `spartan chains list`
- `spartan chains get <chain-id>`
- `spartan chains create --file <path>`
- `spartan chains submit <chain-id>`
- `spartan chains delete <chain-id>`

Chain nodes persist a `request` object that matches the live scrape/crawl/research submission JSON for that node kind. `spartan chains submit --overrides` uses that same per-node request shape.

### Watches

- `spartan watch add --url <url> [flags]`
- `spartan watch list`
- `spartan watch delete <id>`
- `spartan watch check <id>`
- `spartan watch start`

Optional trigger flags on `spartan watch add` let a watch submit a job when change is detected:

- `--trigger-kind scrape|crawl|research`
- `--trigger-request-file <path>` or `--trigger-request-json '{...}'`

The trigger request uses the same operator-facing scrape/crawl/research JSON contract as the live REST job submission endpoints and schedule `request` payloads.

### Schedules

- `spartan schedule add --kind <scrape|crawl|research> --interval <seconds> [job flags]`
- `spartan schedule list`
- `spartan schedule delete --id <id>`

Example:

```bash
spartan schedule add --kind scrape --interval 3600 --url https://example.com
```

API note:

- `/v1/schedules` accepts `kind`, `intervalSeconds`, and a `request` object that matches the live scrape/crawl/research submission contract for the selected kind.

### Export

Supported formats:

- `json`
- `jsonl`
- `csv`
- `md`
- `xlsx`

Direct export:

```bash
spartan export --job-id <id> [flags]
```

Key flags:

- `--format <json|jsonl|csv|md|xlsx>`
- `--out <path>`
- `--schedule-id <export-schedule-id>` to seed format/shape/transform from a persisted recurring export
- `--shape-file <path>` to apply a saved `ExportShapeConfig`
- `--transform-file <path>` to apply a saved `ResultTransformConfig`
- `--transform-expression "..."`
- `--transform-language jmespath|jsonata`

Examples:

```bash
spartan export --job-id 123 --format jsonl --out ./out/results.jsonl
spartan export --job-id 123 --format md --shape-file ./out/report-shape.json --out ./out/report.md
spartan export --job-id 123 --schedule-id <export-schedule-id> --out ./out/projected.csv
spartan export --job-id 123 --format json --transform-language jmespath --transform-expression '{title: title, url: url}'
```

Direct exports use the same bounded `format` / `shape` / `transform` contract as recurring export schedules. `shape` and `transform` remain mutually exclusive.

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
  --destination local

spartan export-schedule add \
  --name "Projected CSV" \
  --filter-kinds scrape \
  --format csv \
  --destination local \
  --transform-language jmespath \
  --transform-expression '{title: title, url: url}'
```

Local destinations default to `exports/{kind}/{job_id}.{format}` when you do not pass `--local-path`.

Recurring exports can persist either:

- a bounded `transform` (`ResultTransformConfig`) for projecting/filtering saved results before export, or
- a bounded `shape` (`ExportShapeConfig`) for field selection/formatting on canonical `md`/`csv`/`xlsx` exports.

Do not combine both on the same schedule; Spartan treats transform and shape as mutually exclusive so recurring exports keep one deterministic projection contract.

For `md`, `csv`, and `xlsx` schedules, the Web UI, REST API, CLI (`spartan ai export-shape`), and MCP (`ai_export_shape`) can generate bounded `ExportShapeConfig` suggestions from a representative job result before you save or update the recurring export. For any schedule format, the Web UI, REST API, CLI (`spartan ai transform`), and MCP (`ai_transform_generate`) can generate or tune a saved `ResultTransformConfig` from a representative job result before you save or update the recurring export.

### Webhook deliveries

Commands:

- `spartan webhook deliveries list [--job-id <job-id>] [--limit <n>] [--offset <n>]`
- `spartan webhook deliveries get <delivery-id>`

Examples:

```bash
spartan webhook deliveries list
spartan webhook deliveries list --job-id <job-id> --limit 50 --offset 0
spartan webhook deliveries get <delivery-id>
```

The CLI prefers the local REST API when it is available and falls back to the persisted delivery store when running serverless or offline. Output is sanitized so webhook credentials, query tokens, and obvious secrets in failure text do not leak to terminal operators.

### Retention, backup, and restore

Retention:

- `spartan retention status`
- `spartan retention cleanup [--dry-run]`

Backup and restore:

- `spartan backup create [-o <dir>] [--exclude-jobs]`
- `spartan backup list [--dir <dir>]`
- `spartan restore --from <archive.tar.gz> [--dry-run] [--force]`
- `spartan reset-data [--backup-dir <dir>] [--force]`

`spartan reset-data` is the operator cutover path for pre-Balanced 1.0 `.data` directories. It archives the full existing data directory to `output/cutover/` by default, recreates `DATA_DIR`, and leaves the next `spartan server` start on a fresh store.

### Service entrypoints

```bash
spartan server
spartan health
spartan proxy-pool status
spartan tui
spartan mcp
spartan version
```

### TUI scope

`spartan tui` remains a lightweight local inspection surface, not a feature-parity or AI authoring surface.

- The TUI is for browsing jobs, statuses, templates, profiles, schedules, and crawl state.
- The TUI may show AI-related job metadata that already exists in persisted job specs or results.
- Dedicated AI preview, AI template generation, AI template debugging, AI render-profile generation, AI render-profile debugging, AI pipeline-JS generation, AI pipeline-JS debugging, AI research refinement, AI export shaping, AI transform generation, and other prompt-heavy authoring flows live in the Web UI, API, CLI (`spartan ai ...`), and MCP (`ai_extract_preview`, `ai_template_generate`, `ai_template_debug`, `ai_render_profile_generate`, `ai_render_profile_debug`, `ai_pipeline_js_generate`, `ai_pipeline_js_debug`, `ai_research_refine`, `ai_export_shape`, `ai_transform_generate`) instead.
- Operator-surface parity work should target Web UI, CLI, and MCP over the shared API/store contracts before considering TUI expansion.
- Do not add TUI-only workflows unless the roadmap explicitly changes this policy.

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

The Settings route includes proxy-pool status inspection plus per-request proxy override/selection controls on the job forms, alongside render profiles, pipeline JS, retention, and other runtime inventory panels.

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
- `/v1/jobs/{id}/export`
- `/v1/jobs/batch/*`
- `/v1/chains*`
- `/v1/watch*`
- `/v1/schedules*`
- `/v1/export-schedules*`
- `/v1/webhooks/deliveries*`
- `/v1/templates*`
- `/v1/ai/*`
- `/v1/render-profiles*`
- `/v1/pipeline-js*`
- `/v1/auth/profiles*`
- `/v1/auth/import`
- `/v1/auth/export`
- `/v1/auth/oauth/*`
- `/v1/ws`

Single-job create/get/cancel flows now share one envelope: create/get return `{ job }`, cancel returns the updated `{ job }` envelope unless `force=true` deletes the record and returns `{ status: "deleted" }`. Job listings return `{ jobs, total, limit, offset }`. Batch create/get/cancel flows now share `{ batch, jobs, total, limit, offset }`, where `batch.stats` is always present and `limit: 0` means individual jobs were not requested.

`GET /v1/jobs/{id}/results` is the raw persisted-results inspection surface (`jsonl`/`json` plus jsonl pagination). `POST /v1/jobs/{id}/export` is the canonical direct export/download surface for `json`, `jsonl`, `md`, `csv`, and `xlsx` with optional bounded `shape` or `transform` controls. `POST /v1/watch/{id}/check` now returns screenshot and visual-diff artifacts as `artifacts[]` descriptors with relative `downloadUrl` values, and `GET /v1/watch/{id}/artifacts/{artifactKind}` is the canonical way to fetch those bytes without exposing host-local paths.

Webhook contract notes:

- Non-export webhook events are delivered as JSON bodies matching the `WebhookPayload` schema in `api/openapi.yaml`.
- `export_completed` deliveries now use one explicit multipart contract everywhere direct/scheduled exports emit them: the request body is `multipart/form-data` with a JSON `metadata` part and an `export` file part containing the rendered export bytes.
- Create/update flows syntax-validate webhook URLs early; runtime delivery separately re-validates the destination, pins outbound dialing to the validated IP set, and refuses redirect hops to a different host.
- Export/job webhook metadata no longer exposes host-local filesystem paths; consumers should use `resultUrl` when they need to fetch persisted job results.

Bounded AI authoring endpoints live under `/v1/ai/*`. Preview/template/render-profile/pipeline request bodies accept optional `images` arrays of request-scoped `{data, mime_type}` attachments, which are used only for the current authoring request and are not persisted as job artifacts:

- `/v1/ai/extract-preview`
- `/v1/ai/template-generate`
- `/v1/ai/template-debug`
- `/v1/ai/render-profile-generate`
- `/v1/ai/render-profile-debug`
- `/v1/ai/pipeline-js-generate`
- `/v1/ai/pipeline-js-debug`
- `/v1/ai/research-refine`
- `/v1/ai/export-shape`

For scrape, crawl, and research job creation, AI extraction rides inside the normal extract payload:

- `extract.ai.enabled`
- `extract.ai.mode`
- `extract.ai.prompt` for natural-language mode
- `extract.ai.schema` for schema-guided mode
- `extract.ai.fields`

Research requests also accept additive bounded agentic controls at the top level:

- `agentic.enabled`
- `agentic.instructions`
- `agentic.maxRounds`
- `agentic.maxFollowUpUrls`

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
- `batch_status`
- `batch_cancel`
- `job_export`
- `export_schedule_list`
- `export_schedule_get`
- `export_schedule_create`
- `export_schedule_update`
- `export_schedule_delete`
- `export_schedule_history`
- `webhook_delivery_list`
- `webhook_delivery_get`

`scrape_page`, `crawl_site`, and `research` now accept the same request bodies as the REST job-submission endpoints for their respective kinds.

Use the same top-level fields you would send to `/v1/scrape`, `/v1/crawl`, or `/v1/research`, including nested execution objects such as:

- `auth`
- `extract`
- `pipeline`
- `webhook`
- `screenshot`
- `device`
- `networkIntercept`
- `agentic` on `research`

That means AI extraction rides inside `extract.ai.*`, proxy transport rides inside `auth.proxy` / `auth.proxyHints`, and screenshot / interception options use the same object shape as REST and the Web UI.

`job_status` and `job_cancel` return the same `{ job }` envelope shape as `GET /v1/jobs/{id}`. `job_list` returns `{ jobs, total, limit, offset }`. `batch_status` and `batch_cancel` now mirror the REST batch envelope `{ batch, jobs, total, limit, offset }`, with optional `includeJobs`, `limit`, and `offset` arguments on the batch tools.

`job_export` accepts the same direct export contract as `spartan export` / `POST /v1/jobs/{id}/export`:

- `format: "jsonl" | "json" | "md" | "csv" | "xlsx"`
- `shape: ExportShapeConfig`
- `transform: ResultTransformConfig`

`shape` and `transform` are mutually exclusive for direct exports just as they are for recurring export schedules.

`job_export` returns an object with:

- `format`
- `filename`
- `contentType`
- `encoding: "utf8" | "base64"`
- `content`

The export-schedule tools use the same persisted `filters`, `export`, and `retry` objects as `/v1/export-schedules*`, including optional `export.transform` and `export.shape` (mutually exclusive).

`webhook_delivery_list` returns `{ deliveries, total, limit, offset }` with optional `jobId`, `limit`, and `offset` arguments. `webhook_delivery_get` returns one sanitized delivery record by id.

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
- The supported path forward is to run `spartan reset-data` or point `DATA_DIR` at a different empty directory.

This is deliberate: the project no longer attempts to open legacy layouts under the reduced 1.0 product boundary.

## pi bridge defaults

Repo-local AI defaults live in `.env` and `config/pi-routes.json`.

- Default pi route order: `kimi-coding/k2p5`, `zai/glm-5`, `openai-codex/gpt-5.4`
- Spartan only passes route IDs to pi; pi continues to own auth, account selection, and billing behavior.
- Override `PI_CONFIG_PATH` or edit `config/pi-routes.json` if you want a different local route order.

## Local CI

Required local gate:

```bash
make verify-toolchain
make ci
```

Useful commands:

```bash
make verify-toolchain
make install
make generate
make build
make test-ci
make ci
make ci-slow
```

`make ci-slow` provisions Playwright and runs the heavier local-fixture/browser validation lane.
