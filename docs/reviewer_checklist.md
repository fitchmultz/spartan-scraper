# Reviewer Validation Checklist

This checklist is designed for a quick, high-confidence review.

## 1) Build + test gate

```bash
git clone <repo-url>
cd spartan-scraper
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

- health endpoint returns `ok: true`
- server logs are clean and actionable

## 4) Web UI smoke check

Terminal A:

```bash
./bin/spartan server
```

Terminal B:

```bash
make web-dev
```

Open <http://localhost:5173>.

Expected:

- UI loads without console/runtime errors
- command palette and onboarding interactions are functional
- API-backed sections render expected empty/loading states

## 5) Public-release hygiene check

```bash
make audit-public
```

Expected:

- no absolute local path leaks
- no placeholder contact strings
- no tracked local/build/cache artifacts
- no branch-history residue for blocked artifact paths

## 6) Optional heavy confidence checks

```bash
make ci-slow
```

Use for network/stress/e2e confidence, not as part of every local inner loop.
