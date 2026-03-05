# Performance and Scaling Notes

This guide summarizes practical tuning levers for Spartan Scraper in local and self-hosted deployments.

## Core Throughput Controls

### Worker concurrency

- `MAX_CONCURRENCY` controls how many jobs can run simultaneously.
- Higher values increase throughput but also increase CPU, memory, and outbound request pressure.

### Request-level timeouts

- `REQUEST_TIMEOUT_SECONDS` sets the default request timeout.
- Shorter timeouts reduce tail latency but can increase false failures on slower sites.

### Rate limiting

- `RATE_LIMIT_QPS` and `RATE_LIMIT_BURST` apply global request throttling.
- Use lower values for fragile targets and stricter compliance requirements.

## Fetch Strategy Trade-offs

### HTTP fetcher (fastest baseline)

Best for static or mostly static content. Lowest resource overhead.

### Chromedp / Playwright (heavier but more resilient)

Best for JS-heavy targets that require runtime rendering or interaction.

- Expect higher CPU and memory usage.
- Prefer targeted enablement per host/profile instead of global headless defaults.

## Payload and Memory Guardrails

- `MAX_RESPONSE_BYTES` caps response payload size.
- Keep this bounded to prevent excessive memory use on large pages.

## Crawl-Scale Controls

Use these flags together to limit crawl blast radius:

- `--max-depth`
- `--max-pages`
- `--same-host-only`
- `--domain-scope`
- include/exclude URL patterns

## Suggested Baseline Profiles

### Small local run

- `MAX_CONCURRENCY=2`
- `RATE_LIMIT_QPS=2`
- `RATE_LIMIT_BURST=4`

### Medium shared environment

- `MAX_CONCURRENCY=4`
- `RATE_LIMIT_QPS=5`
- `RATE_LIMIT_BURST=10`

### Large dedicated worker host

- `MAX_CONCURRENCY=8+` (validate with load tests)
- Tune rate limits by target host constraints
- Prefer profile-based fetch strategy overrides

## Verification Workflow

Use local gates before and after tuning changes:

```bash
make ci
make ci-slow
make ci-network
```

For stress and e2e checks, compare:

- completed jobs per minute
- average job duration
- failure/cancellation rates
- resource usage (CPU, memory)
