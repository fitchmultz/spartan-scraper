# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Treat the Web UI as a first-class operator surface. When parity work and workflow clarity compete, prioritize the product workflow that helps operators complete real tasks faster and with less confusion.
- Prefer feature symmetry across the primary product interfaces that carry the main operator and automation workflows: API, Web UI, CLI, and MCP, but do not preserve a poor Web UI solely for parity.
- Treat the TUI as an intentionally limited local inspection surface, not a feature-parity target, unless this roadmap explicitly says otherwise.
- Add AI enablement where it improves a real scraping, template-authoring, or results-analysis workflow; do not force AI into surfaces where it adds little operational value.
- Preserve route-per-major-feature information architecture at the top level, then use sub-routing or explicit in-route navigation inside complex Web surfaces when that materially improves clarity.
- Treat interface asymmetry as intentional only when this roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.
- Put meaningful operator-facing product work ahead of maintenance, cleanup, and policy reminders.
- Treat focused failure-path dogfooding as acceptance criteria for major operator workflow cutovers, not as a standalone roadmap epic.

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level feature routes (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), while treating `/automation/:section` and `/settings/:section` as the canonical deep-link shapes for those sectioned surfaces.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.

## Audit-Derived Follow-up Work (2026-03-30)

Audit snapshot: 156 non-test code files exceed 300 lines, the current Go/TS heuristics found roughly 393 functions over 50 LOC, and 62 tracked code files still miss the required top-of-file purpose header.

## Manual QA Follow-up Batches (2026-03-31)

### Batch 1 — Operator action visibility and dead-end removal

- Keep the primary commit/submit controls visible on long authoring surfaces instead of hiding them below large forms or assistant rails. The immediate cutover fixed the template workspace plus watch/export dialogs, but the remaining results-route export chooser still needs the same treatment.
- Ensure every first-run operator path on a 1280×720 laptop-height viewport exposes a visible next action without requiring guesswork about whether the route is blocked or simply farther down the page.
- Accept this batch only after a fresh manual dogfood pass covers: job detail → export chooser, promoted template authoring, watch creation, and export schedule creation.

### Batch 2 — Promotion guidance for blank template drafts

- Make the “promote to template” path explain blank-draft saves more directly when Spartan cannot safely infer reusable selectors from the source job.
- Keep save affordances disabled with explicit inline reasons until the minimum valid selector set exists, and make the first reusable rule easier to author from the seeded draft.
- Dogfood acceptance: start from a plain scrape job, promote to a template draft, and reach a successful save without trial-and-error.

### Batch 3 — Jobs and results surface density cleanup

- Reduce the amount of vertical scanning required before completed-job actions, promotion actions, and export controls become obvious on common laptop viewport heights.
- Revisit the balance between summary cards, status lanes, and action rails so core “inspect / export / promote” tasks stay above the fold more often after the current Web cutover.
- Dogfood acceptance: repeated route hopping between `/jobs`, `/jobs/:id`, `/templates`, and `/automation/*` should feel fast and unsurprising on both desktop and mobile-width layouts.


