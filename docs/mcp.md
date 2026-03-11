# MCP Guide

## Supported MCP Role

- MCP remains the agent-facing control plane for the 1.0 core.
- Submit scrape, crawl, and research jobs.
- Poll status, fetch results, and export supported formats.

## Long-Running Jobs

- Treat MCP jobs as asynchronous.
- Submit, store the returned job ID, then poll or wait for terminal status.
- Use the job manifest on disk when you need artifact-level inspection.
