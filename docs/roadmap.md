# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Recently Completed

- Webhook delivery inspection now spans API, Web UI, CLI, and MCP with sanitized URL/error output, so operators can debug retries and failures from `spartan webhook deliveries ...` and `webhook_delivery_list` / `webhook_delivery_get` without reading host-local files.

## Now

- Add TUI inspection flows for webhook deliveries, export schedule history, and watch-trigger outcomes so terminal-only operators can debug automation without leaving the TUI.
- Close the remaining highest-value operator-surface gaps across API, Web UI, CLI, MCP, and TUI for export schedules, watches, and batches in that order.
- Expand operator observability so run history, export outcomes, and delivery failures are actionable without digging through host-local files or internal state.
