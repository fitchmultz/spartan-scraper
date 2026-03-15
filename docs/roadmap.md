# Roadmap

This is the canonical source of truth for planned work, exploratory ideas, and sequencing.

## Planning Principles

- Prefer feature symmetry across the primary product interfaces: API, Web UI, CLI, MCP, and TUI where the capability is meaningful in that interface.
- Add AI enablement where it improves a real scraping/research workflow; do not force AI into surfaces where it adds little operational value.
- Treat interface asymmetry as intentional only when the roadmap says so explicitly.
- Prefer roadmap ordering that limits churn in shared contracts, generated clients, and operator-facing docs.

## Now

- Audit remaining operator-facing execution flags and runtime bridges so every persisted execution option either threads end-to-end or is removed before more automation surfaces expand.
- Unify chain node submission and watch-triggered job creation on the same operator-facing execution model used by live jobs and schedules, eliminating the remaining contract-special cases before adding more automation surfaces.
- Normalize job and batch response shaping across REST, Web UI, and MCP so downstream clients can consume one stable status/result envelope without transport-specific branching.
- Remove raw screenshot and diff filesystem paths from watch check API responses, replace them with artifact handles or download endpoints, and regenerate the OpenAPI/client/docs contract so public responses stop advertising host-local paths.
- Either pin webhook connections to the prevalidated IP set during dialing or narrow the SSRF / DNS-rebinding claim everywhere it appears; add resolver-controlled integration coverage for whichever guarantee the product actually intends to make.
- Remove the temporary transitive Go dependency overrides in `go.mod` as soon as upstream parent modules absorb those newer tags; keep `make audit-deps` green so override cleanup happens immediately once the parent graph catches up.
