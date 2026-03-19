# Validation Checklist

This checklist is designed for a quick, high-confidence validation pass.

## 1) Build + test gate

```bash
git clone <repo-url>
cd spartan-scraper
make verify-toolchain
make ci-pr
```

Expected:

- command exits `0`
- no warnings/errors in lint/type-check/tests
- no generated/format drift introduced by CI steps

## 2) CLI smoke check

```bash
./bin/spartan --help
./bin/spartan version
```

Expected:

- help shows command tree
- version output includes build metadata

## 3) API server smoke check

Terminal A:

```bash
./bin/spartan server
```

Terminal B:

```bash
curl -sS http://127.0.0.1:8741/healthz
```

Expected:

- health endpoint returns JSON with `"status":"ok"`
- server logs are clean and actionable
- optional AI/proxy/retention components may appear as `disabled` without turning a default first run into a failure
- if you launch with `BIND_ADDR=0.0.0.0`, non-health API routes require `X-API-Key`

## 4) WebSocket origin safety check

Terminal A:

```bash
./bin/spartan server
```

Terminal B:

```bash
# blocked: non-loopback browser Origin
curl -i -H "Origin: https://example.com" http://127.0.0.1:8741/v1/ws
```

Expected:

- response is `403 Forbidden`
- body includes `forbidden websocket origin`

## 5) Web UI smoke check

Terminal A:

```bash
./bin/spartan server
```

Terminal B:

```bash
make web-dev
# If the backend is not on 8741:
# DEV_API_PROXY_TARGET=http://127.0.0.1:<port> make web-dev
```

Open <http://localhost:5173>.

Expected:

- UI loads without console/runtime errors
- command palette and onboarding interactions are functional
- API-backed sections render expected empty/loading states
- a fresh first run stays calm and usable with AI, proxy pooling, and retention left off by default
- batch forms submit successfully in all three modes (`Batch scrape`, `Batch crawl`, `Batch research`) without runtime crashes

## 6) Release-hygiene checks

```bash
make audit-public
```

Expected:

- no absolute local path leaks
- no placeholder contact strings
- no high-confidence secret patterns
- no tracked local/build/cache artifacts
- no branch-history residue for blocked artifact paths

## 7) Optional deep history secret scan

```bash
make secret-scan
```

Expected:

- no unreviewed high-confidence secrets in full git history
- scan succeeds using the committed `.gitleaksignore` baseline

## 8) Optional heavy confidence checks

```bash
make ci-slow
```

Use for deterministic stress/e2e confidence, not as part of every local inner loop. The target provisions Playwright automatically so the result is meaningful on a clean machine. Run `make ci-network` separately if you want live-Internet smoke validation.
