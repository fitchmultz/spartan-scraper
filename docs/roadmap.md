# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces that carry the main operator and automation workflows: API, Web UI, CLI, and MCP.
- Treat the TUI as an intentionally limited local inspection surface, not a feature-parity target, unless this roadmap explicitly says otherwise.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Recently Completed

- Watch management now spans API, Web UI, CLI, and MCP with `watch_list`, `watch_get`, `watch_create`, `watch_update`, `watch_delete`, and `watch_check`, so operators and agents can manage stored watches and run manual checks without falling back to REST-only flows.
- Webhook delivery inspection now spans API, Web UI, CLI, and MCP with sanitized URL/error output, so operators can debug retries and failures from `spartan webhook deliveries ...` and `webhook_delivery_list` / `webhook_delivery_get` without reading host-local files.

## Now

- Close the remaining highest-value operator-surface gaps across API, Web UI, CLI, and MCP for batches.
- Expand operator observability so run history, export outcomes, watch outcomes, and delivery failures are actionable without digging through host-local files or internal state.
- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item justifies re-investing in it as a first-class surface.
