# Security Model

## Supported Boundary

- Single trusted operator deployment only.
- No multi-user isolation guarantees.
- No distributed trust boundary.

## Authentication

- Loopback usage can run without API keys.
- Non-loopback binds must use API keys.
- Target-site credentials belong in the auth vault, not in job specs checked into source control.

## Bind Behavior

```bash
spartan server
BIND_ADDR=0.0.0.0 spartan server
```

The first form is loopback-only local usage. The second exposes the API off-host and must be protected with API keys.

## API Key Example

```bash
export SPARTAN_API_KEY='set-a-local-key'
curl --config <(printf 'header = "X-API-Key: %s"\n' "$SPARTAN_API_KEY") \
  http://127.0.0.1:8741/v1/jobs
```

## Auth Vault Expectations

- Keep target-site credentials in `.data/auth_vault.json` via auth profiles.
- Use `authProfile` in job or schedule specs when you want profile-based auth resolution at execution time.
- Do not commit vault contents, API keys, or live target credentials.

## Redaction

- Job errors and exposed job specs are redacted before API and MCP responses.
- Filesystem paths are removed from public-facing job views.
