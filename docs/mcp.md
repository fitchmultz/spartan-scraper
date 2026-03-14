# MCP Guide

## Supported MCP Role

- MCP remains the agent-facing control plane for the 1.0 core.
- Submit scrape, crawl, and research jobs.
- Poll status, fetch results, and export supported formats.

## Long-Running Jobs

- Treat MCP jobs as asynchronous.
- Submit, store the returned job ID, then poll or wait for terminal status.
- Use the job manifest on disk when you need artifact-level inspection.

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

## Example

```json
{"id":1,"method":"initialize"}
{"id":2,"method":"tools/call","params":{"name":"research","arguments":{"query":"pricing model","urls":["https://example.com/pricing","https://example.com/support"],"aiExtract":true,"aiMode":"natural_language","aiPrompt":"Extract the pricing model, contract terms, and support commitments","aiFields":["pricing_model","contract_terms","support_commitments"],"agentic":true,"agenticInstructions":"Prioritize pricing and support commitments","agenticMaxRounds":2,"agenticMaxFollowUpUrls":4}}}
{"id":3,"method":"tools/call","params":{"name":"job_status","arguments":{"id":"<job-id>"}}}
{"id":4,"method":"tools/call","params":{"name":"job_results","arguments":{"id":"<job-id>"}}}
```

The expected pattern is submit, capture the returned job ID, poll or wait for terminal status, then fetch results or export.
