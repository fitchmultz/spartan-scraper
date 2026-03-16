# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces that carry the main operator and automation workflows: API, Web UI, CLI, and MCP.
- Treat the TUI as an intentionally limited local inspection surface, not a feature-parity target, unless this roadmap explicitly says otherwise.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.
- Put meaningful operator-facing product work ahead of maintenance, cleanup, and policy reminders.

## Recently Completed

- Run history observability now spans API, Web UI, CLI, and MCP with recent execution inspection, explicit batch queue progression, recent failed-run views, and structured failure context so operators can understand outcomes without digging through host-local files or internal state.
- Watch management now spans API, Web UI, CLI, and MCP with `watch_list`, `watch_get`, `watch_create`, `watch_update`, `watch_delete`, and `watch_check`, so operators and agents can manage stored watches and run manual checks without falling back to REST-only flows.
- Webhook delivery inspection now spans API, Web UI, CLI, and MCP with sanitized URL/error output, so operators can debug retries and failures from `spartan webhook deliveries ...` and `webhook_delivery_list` / `webhook_delivery_get` without reading host-local files.
- Batch management now spans API, Web UI, CLI, and MCP with authoritative list/detail/cancel flows plus MCP batch submission tools, so operators and agents can create, enumerate, inspect, and stop persisted batches without falling back to REST-only workflows or browser-local tracking.

## Now

- Add export outcome inspection so direct and scheduled export successes/failures can be audited consistently across API, Web UI, CLI, and MCP.
- Add watch outcome and check-history inspection so stored monitoring workflows surface recent results, diffs, and failures coherently across the primary operator surfaces.

## After

- Keep docs and examples aligned with each observability cutover so the canonical operator workflow stays current as contracts and surfaces evolve.
- Polish cross-surface contract consistency discovered during observability work, including pagination, filters, envelope naming, and generated client/doc sync where needed.

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
