# MCP Guide

## Supported MCP Role

- MCP remains the agent-facing control plane for the 1.0 core.
- Submit scrape, crawl, and research jobs.
- Poll status, fetch results, and export supported formats.

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

These tools return structured authoring results immediately and do not create jobs. Attached `images` are bounded, request-scoped visual context only and are not persisted as job artifacts.

## AI extraction arguments

`scrape_page`, `crawl_site`, and `research` support the same AI extraction controls as the direct job-submission surfaces:

- `aiExtract: true`
- `aiMode: "natural_language" | "schema_guided"`
- `aiPrompt: "..."` for natural-language mode
- `aiSchema: { ... }` for schema-guided mode
- `aiFields: ["field1", "field2"]`

`research` also supports bounded agentic follow-up controls:

- `agentic: true`
- `agenticInstructions: "..."`
- `agenticMaxRounds: 1..3`
- `agenticMaxFollowUpUrls: 1..10`

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
{"id":11,"method":"tools/call","params":{"name":"research","arguments":{"query":"pricing model","urls":["https://example.com/pricing","https://example.com/support"],"aiExtract":true,"aiMode":"natural_language","aiPrompt":"Extract the pricing model, contract terms, and support commitments","aiFields":["pricing_model","contract_terms","support_commitments"],"agentic":true,"agenticInstructions":"Prioritize pricing and support commitments","agenticMaxRounds":2,"agenticMaxFollowUpUrls":4}}}
{"id":12,"method":"tools/call","params":{"name":"proxy_pool_status","arguments":{}}}
{"id":13,"method":"tools/call","params":{"name":"job_status","arguments":{"id":"<job-id>"}}}
{"id":14,"method":"tools/call","params":{"name":"job_results","arguments":{"id":"<job-id>"}}}
```

The expected pattern is: use the dedicated AI authoring tools when you want immediate preview/template/configuration/refinement/export-shape output, and use the job tools when you need persisted scrape/crawl/research execution that can be polled, exported, and inspected later.
