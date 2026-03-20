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

## Next

1. AI Template Validation Flexibility
   - Align bridge-side template validation with the real template model used by Spartan: valid generated templates may be selector-driven, JSON-LD-driven, regex-driven, or mixed.
   - Remove the selector-only assumption from bridge validation while preserving strict checks that templates still have a name and at least one real extraction rule.
   - Keep downstream validation strict about malformed selectors, malformed regex rules, invalid JSON-LD paths, or templates that cannot pass local structural validation.
   - Add targeted tests for:
     - JSON-LD-only template generation
     - regex-only template generation
     - mixed templates
     - still-invalid empty templates
   - Ensure error messages explain the real invariant being enforced rather than implying selectors are the only valid extraction strategy.

1. Optional Goal Defaults for AI Automation Generators
   - Let render-profile and pipeline-JS generation bootstrap from page context when explicit operator instructions are omitted, instead of hard-failing before the model can attempt a reasonable starter configuration.
   - Preserve explicit operator guidance as the preferred path, but provide a sensible default objective derived from the page URL, fetched HTML, JS-heaviness signals, and any attached screenshots.
   - Keep deterministic validation strict after generation:
     - generated render profiles must still pass structural validation and recheck where applicable
     - generated pipeline scripts must still pass structural validation and representative execution checks where applicable
   - Update API, CLI, MCP, and Web copy so these flows are described as “instructions optional, guidance recommended” rather than “instructions required” when the product can succeed without them.
   - Add tests for no-instruction starter generation plus existing tests for explicitly guided generation.
   - Sequence this after the bridge hardening work so default-goal generation benefits from the more tolerant bridge behavior instead of inheriting the current brittle path.

## Ongoing Constraints

- Keep the TUI scope frozen as a lightweight local inspector unless a future roadmap item explicitly justifies re-investing in it as a first-class surface.
- Preserve the current top-level Web route model (`/jobs`, `/jobs/new`, `/jobs/:id`, `/templates`, `/automation`, and `/settings`), but allow sub-routing or in-route navigation inside complex surfaces when that materially improves operator comprehension.
- Treat Web UI product-grade workflow improvements as higher priority than maintenance-only work and parity-only work until the primary operator journeys feel coherent, legible, and fast.
- No backwards-compatibility shims are required for the Web UI cutover. Prefer the cleaner immediate product model when redesign choices conflict.
