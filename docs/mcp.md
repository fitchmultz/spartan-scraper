# MCP Guide

## Supported MCP Role

- MCP remains the agent-facing control plane for the 1.0 core.
- Submit scrape, crawl, and research jobs.
- Poll status, fetch results, and export supported formats.

## Long-Running Jobs

- Treat MCP jobs as asynchronous.
- Submit, store the returned job ID, then poll or wait for terminal status.
- Use the job manifest on disk when you need artifact-level inspection.

## Example

```json
{"id":1,"method":"initialize"}
{"id":2,"method":"tools/call","params":{"name":"scrape_page","arguments":{"url":"https://example.com","timeoutSeconds":30}}}
{"id":3,"method":"tools/call","params":{"name":"job_status","arguments":{"id":"<job-id>"}}}
{"id":4,"method":"tools/call","params":{"name":"job_results","arguments":{"id":"<job-id>"}}}
```

The expected pattern is submit, capture the returned job ID, poll or wait for terminal status, then fetch results or export.
