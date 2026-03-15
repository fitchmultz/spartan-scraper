# MCP Guide

## Supported MCP Role

- MCP remains the agent-facing control plane for the 1.0 core.
- Submit scrape, crawl, and research jobs.
- Poll status, fetch results, export supported text formats, and manage export schedules.

## Long-Running Jobs

- Treat MCP jobs as asynchronous.
- Submit, store the returned job ID, then poll or wait for terminal status.
- Use the job manifest on disk when you need artifact-level inspection.

## AI authoring tools

MCP exposes dedicated prompt-heavy AI authoring tools in addition to job submission:

- `ai_extract_preview`
  - `images: [{"data":"<base64>","mime_type":"image/png"}]` optional request-scoped reference images
  - `url` or `html`
  - `mode: "natural_language" | "schema_guided"`
  - `prompt: "..."` for natural-language mode
  - `schema: { ... }` for schema-guided mode
  - `fields: ["field1", "field2"]`
  - `headless: true|false`
  - `playwright: true|false`
  - `visual: true|false` to capture a screenshot and send multimodal visual context when fetching a URL
- `ai_template_generate`
  - `images: [{"data":"<base64>","mime_type":"image/png"}]` optional request-scoped reference images
  - `url` or `html`
  - `description: "..."`
  - `sampleFields: ["field1", "field2"]`
  - `headless: true|false`
  - `playwright: true|false`
  - `visual: true|false` to capture a screenshot and send multimodal visual context when fetching a URL
- `ai_template_debug`
  - `images: [{"data":"<base64>","mime_type":"image/png"}]` optional request-scoped reference images
  - `url` or `html`
  - `template: { ... }`
  - `instructions: "..."`
  - `headless: true|false`
  - `playwright: true|false`
  - `visual: true|false`
- `ai_render_profile_generate`
  - `images: [{"data":"<base64>","mime_type":"image/png"}]` optional request-scoped reference images
  - `url`
  - `instructions: "..."`
  - `name: "..."` optional
  - `hostPatterns: ["example.com", "*.example.com"]` optional
  - `headless: true|false`
  - `playwright: true|false`
  - `visual: true|false`
- `ai_render_profile_debug`
  - `images: [{"data":"<base64>","mime_type":"image/png"}]` optional request-scoped reference images
  - `url`
  - `profile: { ... }`
  - `instructions: "..."` optional
  - `headless: true|false`
  - `playwright: true|false`
  - `visual: true|false`
- `ai_pipeline_js_generate`
  - `images: [{"data":"<base64>","mime_type":"image/png"}]` optional request-scoped reference images
  - `url`
  - `instructions: "..."`
  - `name: "..."` optional
  - `hostPatterns: ["example.com", "*.example.com"]` optional
  - `headless: true|false`
  - `playwright: true|false`
  - `visual: true|false`
- `ai_pipeline_js_debug`
  - `images: [{"data":"<base64>","mime_type":"image/png"}]` optional request-scoped reference images
  - `url`
  - `script: { ... }`
  - `instructions: "..."` optional
  - `headless: true|false`
  - `playwright: true|false`
  - `visual: true|false`
- `ai_research_refine`
  - `result: { ... }` existing `ResearchResult`
  - `instructions: "..."` optional rewrite guidance
- `ai_export_shape`
  - `jobId: "..."` representative completed job whose persisted result should seed the shape
  - `format: "md" | "csv" | "xlsx"`
  - `currentShape: { ... }` optional existing `ExportShapeConfig`
  - `instructions: "..."` optional field-selection guidance
- `ai_transform_generate`
  - `jobId: "..."` representative completed job whose persisted result should seed the transform
  - `currentTransform: { expression: "...", language: "jmespath" | "jsonata" }` optional existing transform to tune
  - `preferredLanguage: "jmespath" | "jsonata"` optional preferred output language
  - `instructions: "..."` optional projection/filter guidance

These tools return structured authoring results immediately and do not create jobs. Attached `images` are bounded, request-scoped visual context only and are not persisted as job artifacts.

## Export tools

- `job_export`
  - `id: "..."`
  - `format: "jsonl" | "json" | "md" | "csv" | "xlsx"` optional, defaults to `jsonl`
  - `shape: { ... }` optional `ExportShapeConfig` for markdown/tabular exports
  - `transform: { expression: "...", language: "jmespath" | "jsonata" }` optional saved-result projection/filter
  - returns `{ format, filename, contentType, encoding, content }` where `encoding` is `utf8` for text exports and `base64` for `xlsx`
- `export_schedule_list`
  - no arguments
- `export_schedule_get`
  - `id: "..."`
- `export_schedule_create`
  - `name: "..."`
  - `filters: { ... }` persisted `ExportFilters`
  - `export: { ... }` persisted `ExportConfig` including optional `shape` or `transform`
  - `enabled: true|false` optional
  - `retry: { ... }` optional
- `export_schedule_update`
  - `id: "..."`
  - `name: "..."`
  - `filters: { ... }`
  - `export: { ... }`
  - `enabled: true|false` optional
  - `retry: { ... }` optional
- `export_schedule_delete`
  - `id: "..."`
- `export_schedule_history`
  - `id: "..."`
  - `limit: number` optional
  - `offset: number` optional

Direct `job_export` calls and recurring export schedules can persist either `transform` / `export.transform` or `shape` / `export.shape`, but not both. Spartan enforces that mutual exclusion so ad hoc and recurring exports keep one deterministic projection contract.

## Job submission arguments

`scrape_page`, `crawl_site`, and `research` now take the same request objects as `/v1/scrape`, `/v1/crawl`, and `/v1/research`.

That means:

- AI extraction lives under `extract.ai.*`
- proxy transport lives under `auth.proxy` / `auth.proxyHints`
- screenshot capture uses `screenshot`
- device emulation uses `device`
- network interception uses `networkIntercept`
- bounded research follow-up uses `agentic` on `research`

Example nested fields:

- `extract.ai.enabled`
- `extract.ai.mode`
- `extract.ai.prompt` or `extract.ai.schema`
- `pipeline.preProcessors` / `pipeline.postProcessors` / `pipeline.transformers`
- `auth.proxy.url` or `auth.proxyHints.preferred_region`
- `screenshot.enabled`
- `networkIntercept.enabled`

## Examples

```json
{"id":1,"method":"initialize"}
{"id":2,"method":"tools/call","params":{"name":"ai_extract_preview","arguments":{"url":"https://example.com/product","mode":"natural_language","prompt":"Extract the title, price, and availability","fields":["title","price","availability"],"images":[{"mime_type":"image/png","data":"<base64>"}],"headless":true,"visual":true}}}
{"id":3,"method":"tools/call","params":{"name":"ai_template_generate","arguments":{"html":"<html><body><h1>Widget</h1></body></html>","description":"Extract the product title"}}}
{"id":4,"method":"tools/call","params":{"name":"ai_template_debug","arguments":{"url":"https://example.com/product","template":{"name":"product","selectors":[{"name":"title","selector":".missing","attr":"text"}]},"instructions":"Prefer the visible h1","headless":true,"visual":true}}}
{"id":5,"method":"tools/call","params":{"name":"ai_render_profile_generate","arguments":{"url":"https://example.com/app","instructions":"Wait for the dashboard shell and prefer headless mode","visual":true}}}
{"id":6,"method":"tools/call","params":{"name":"ai_render_profile_debug","arguments":{"url":"https://example.com/app","profile":{"name":"example-app","hostPatterns":["example.com"],"wait":{"mode":"selector","selector":".missing"}},"instructions":"Prefer a stable selector wait","visual":true}}}
{"id":7,"method":"tools/call","params":{"name":"ai_pipeline_js_generate","arguments":{"url":"https://example.com/app","instructions":"Wait for the dashboard shell and reset scroll position","visual":true}}}
{"id":8,"method":"tools/call","params":{"name":"ai_pipeline_js_debug","arguments":{"url":"https://example.com/app","script":{"name":"example-app","hostPatterns":["example.com"],"selectors":[".missing"]},"instructions":"Prefer selector waits over post-nav JS","visual":true}}}
{"id":9,"method":"tools/call","params":{"name":"ai_research_refine","arguments":{"result":{"query":"pricing model","summary":"Original research summary","evidence":[{"url":"https://example.com/pricing","title":"Pricing","snippet":"Contact sales for enterprise pricing.","citationUrl":"https://example.com/pricing"}],"citations":[{"canonical":"https://example.com/pricing","url":"https://example.com/pricing"}]},"instructions":"Condense this into an operator-ready brief"}}}
{"id":10,"method":"tools/call","params":{"name":"ai_export_shape","arguments":{"jobId":"<job-id>","format":"md","instructions":"Prioritize summary and pricing fields for operator handoff"}}}
{"id":11,"method":"tools/call","params":{"name":"research","arguments":{"query":"pricing model","urls":["https://example.com/pricing","https://example.com/support"],"auth":{"proxyHints":{"preferred_region":"us-east","required_tags":["residential"]}},"extract":{"ai":{"enabled":true,"mode":"natural_language","prompt":"Extract the pricing model, contract terms, and support commitments","fields":["pricing_model","contract_terms","support_commitments"]}},"agentic":{"enabled":true,"instructions":"Prioritize pricing and support commitments","maxRounds":2,"maxFollowUpUrls":4}}}}
{"id":12,"method":"tools/call","params":{"name":"job_export","arguments":{"id":"<job-id>","format":"json","transform":{"expression":"{title: title, url: url}","language":"jmespath"}}}}
{"id":13,"method":"tools/call","params":{"name":"export_schedule_create","arguments":{"name":"Projected Export","filters":{"job_kinds":["scrape"]},"export":{"format":"csv","destination_type":"local","transform":{"expression":"{title: title, url: url}","language":"jmespath"}}}}}
{"id":14,"method":"tools/call","params":{"name":"proxy_pool_status","arguments":{}}}
{"id":15,"method":"tools/call","params":{"name":"job_status","arguments":{"id":"<job-id>"}}}
{"id":16,"method":"tools/call","params":{"name":"batch_status","arguments":{"id":"<batch-id>","includeJobs":true,"limit":50,"offset":0}}}
{"id":17,"method":"tools/call","params":{"name":"job_results","arguments":{"id":"<job-id>"}}}
```

`job_status` and `job_cancel` now return the same `{ job }` envelope shape as REST job detail. `job_list` returns `{ jobs, total, limit, offset }`. `batch_status` and `batch_cancel` return the same `{ batch, jobs, total, limit, offset }` envelope shape as REST batch create/get/cancel, including `batch.stats` on every response.

The expected pattern is: use the dedicated AI authoring tools when you want immediate preview/template/configuration/refinement/export-shape/transform output, use `job_*` and `batch_*` tools when you need persisted scrape/crawl/research execution that can be polled or canceled later, and use `export_schedule_*` tools when you want recurring export contracts persisted alongside the rest of the runtime control plane.
