# API Guide

## Stable Surfaces

- REST is the canonical integration API.
- WebSocket is the live event channel.
- Job and batch control-plane responses use stable envelopes: single-job endpoints return `{ job }`, job listings return `{ jobs, total, limit, offset }`, and batch create/get/cancel endpoints return `{ batch, jobs, total, limit, offset }`.
- Watch check responses expose persisted screenshot and diff artifacts through explicit `artifacts[].downloadUrl` links; public watch and crawl-state responses do not advertise host-local filesystem paths.
- Webhook URLs are syntax-validated on create/update. Outbound delivery then resolves the target host once per request, pins dialing to that validated IP set, and treats redirect responses as failures instead of following them to a new host.

## Core Workflow

1. Submit `scrape`, `crawl`, or `research`.
2. Read the returned `{ job }` envelope, then poll `/v1/jobs/:id` or subscribe to `/v1/ws`.
3. Read `/v1/jobs/:id/results`.
4. Inspect `.data/jobs/<job-id>/manifest.json` for artifact metadata.

## Scrape Example

```bash
curl -sS http://127.0.0.1:8741/v1/scrape \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://example.com",
    "headless": false,
    "timeoutSeconds": 30
  }'
```

## Crawl Example

```bash
curl -sS http://127.0.0.1:8741/v1/crawl \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://example.com",
    "maxDepth": 1,
    "maxPages": 10,
    "headless": false,
    "timeoutSeconds": 30
  }'
```

## Research Example

```bash
curl -sS http://127.0.0.1:8741/v1/research \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "pricing",
    "urls": ["https://example.com", "https://example.com/docs"],
    "maxDepth": 1,
    "maxPages": 10,
    "headless": false,
    "timeoutSeconds": 30
  }'
```

## Schedule Example

```bash
curl -sS http://127.0.0.1:8741/v1/schedules \
  -H 'Content-Type: application/json' \
  -d '{
    "kind": "scrape",
    "intervalSeconds": 3600,
    "request": {
      "url": "https://example.com",
      "headless": false,
      "timeoutSeconds": 30
    }
  }'
```

## WebSocket Example

```bash
websocat ws://127.0.0.1:8741/v1/ws
```

Job lifecycle messages arrive as JSON envelopes like:

```json
{"type":"job_completed","timestamp":1741712400000,"payload":{"jobId":"<id>","kind":"scrape","status":"succeeded","updatedAt":1741712400000}}
```

## Removed Surfaces

- No GraphQL
- No replay endpoints
- No workspace or multi-user endpoints
