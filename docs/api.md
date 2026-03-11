# API Guide

## Stable Surfaces

- REST is the canonical integration API.
- WebSocket is the live event channel.
- Job responses expose typed `spec` plus `specVersion`.

## Core Workflow

1. Submit `scrape`, `crawl`, or `research`.
2. Poll `/v1/jobs/:id` or subscribe to `/v1/ws`.
3. Read `/v1/jobs/:id/results`.
4. Inspect `.data/jobs/<job-id>/manifest.json` for artifact metadata.

## Removed Surfaces

- No GraphQL
- No replay endpoints
- No workspace or multi-user endpoints
