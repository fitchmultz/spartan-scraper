# Verified Job Promotion Contract Audit

**Status:** Completed planning input  
**Primary surfaces:** `GET /v1/jobs/{id}`, `internal/submission`, `/jobs/:id`, `/templates`, `/automation/watches`, `/automation/exports`

## Summary

This audit resolves the last major planning ambiguity before the verified job promotion cutover.

The platform already stores enough source-job context to seed meaningful promotion flows, but the fit varies sharply by destination. Template promotion has a clean path when reusable extraction rules already exist. Watch promotion is only a partial match to the current watch contract. Export schedule promotion is the weakest conceptual fit and must be framed as future-job export automation, not rerunning the source job.

The audit also confirms that the Web UI should not rely on unsanitized job detail. If browser-visible source data is insufficient, the clean answer is a narrow server-generated promotion-draft contract, not a weaker sanitization policy.

## Problems This Solves

- Prevents the cutover from being built on incorrect assumptions about what the source job already exposes.
- Clarifies which promotion destinations are straightforward, partial, or conceptually mismatched.
- Avoids weakening the API's redaction guarantees just to make promotion easier.
- Forces the first implementation cut to stay aligned with real platform contracts instead of optimistic UI-only heuristics.
- Makes the roadmap implementation-ready without jumping into code before the product model is sound.

## Audit Findings

### 1. Source-job data already exists, but at two different trust levels

The server persists a rich typed job spec for each job kind, and `submission.RequestFromTypedSpec(...)` can reconstruct the operator-facing request from that trusted persisted spec.

That is the strongest existing reverse bridge for promotion planning.

However, the public `GET /v1/jobs/{id}` response returns a sanitized `JobResponse`. `model.SanitizeJob(...)` redacts secrets, auth material, cookies, headers, tokens, passwords, and host-local paths before the job leaves the server boundary.

That means:

- the browser already has a safe source-job contract
- the browser does **not** have a full-fidelity original request contract
- promotion must tolerate redaction or use a new curated draft endpoint if a destination genuinely needs more than the sanitized job can provide

### 2. The authoritative job-detail endpoint already exists, but the Web UI is not using it as the route source of truth

The API already exposes `GET /v1/jobs/{id}` and the generated web client already includes `getV1JobsById`.

The current `/jobs/:id` route in `web/src/App.tsx` resolves `detailJob` only from the current paged jobs list. That is sufficient for read-only list continuity, but it is not sufficient for promotion because the route can be opened for a job that is not currently loaded in the active jobs page or filter.

So the first implementation cut should treat direct job-detail fetch by ID as a required route-level fallback, not a later cleanup.

### 3. Template promotion is viable only when the source job already contains reusable extraction structure

Template creation accepts a narrow extraction-only payload:

- `name`
- `selectors`
- `jsonld`
- `regex`
- `normalize`

That is not a full job recipe.

The good news is that the current job request contract already has two high-confidence reuse paths:

- `extract.template` — a named reusable template already exists and can be loaded through the existing template detail API, then duplicated into a new draft
- `extract.inline` — the source job already contains inline `extract.Template` rules that closely match template payload fields and can seed a draft directly

The weak paths are:

- AI extraction without reusable rule structure
- validation-only or runtime-only extraction configuration
- jobs that never used template-like extraction rules at all

For those cases, template promotion should not fake a full conversion. It should open a guided blank or AI-assisted template draft with clear source context.

### 4. Watch promotion is only a partial fit to the current watch model

The current watch contract supports a single monitored URL plus watch-specific monitoring settings such as:

- `url`
- `intervalSeconds`
- `headless`
- `usePlaywright`
- `diffFormat`
- `screenshotConfig`
- `screenshotEnabled`
- `visualDiffThreshold`
- `notifyOnChange`
- `jobTrigger`

This overlaps with only part of a successful job.

Confirmed reusable overlap from a typical scrape job:

- target URL
- headless / Playwright choice
- some screenshot-related settings when the source job used them

Confirmed non-overlap in the current watch contract:

- no `authProfile`
- no general `auth` payload
- no pipeline config
- no extraction template payload
- no multi-URL or multi-page monitoring model

This means watch promotion cannot honestly behave like “save this job as a watch.”

The first cut should therefore be:

- scrape-first
- single-target
- explicit about which fields were preserved
- explicit about missing watch-only decisions such as interval, notifications, diff sensitivity, and selector targeting

There is also an additional semantic question around `jobTrigger`. A watch can optionally submit a raw job request after change detection, and that trigger is validated through the same canonical submission path as live jobs. That is powerful, but it is **not** the same thing as the watch’s own monitoring configuration. The first promotion cut should decide deliberately whether it seeds only the monitoring draft or also offers an explicit “run a job on change” follow-up, rather than wiring both automatically.

### 5. Export schedule promotion is export automation, not job replay

The export schedule contract is filter-based:

- it matches future completed jobs by metadata (`job_kinds`, `job_status`, `tags`, `has_results`)
- it defines where and how those results should be exported
- it does **not** rerun the source job on a cadence

That makes export schedule promotion the weakest “clone from job” fit.

What a source job can confidently provide:

- a suggested human-readable name
- a likely `job_kind` filter
- a likely successful-status / has-results filter baseline
- possibly an export format or related export intent **only if** the promotion is launched from a result export context that already knows what the operator exported

What a source job does **not** provide by itself:

- a meaningful export destination
- transform / shape policy
- retry intent
- all future-job matching rules
- any real notion of rerun cadence

So export schedule promotion should be framed as “create an ongoing export policy for future matching jobs,” not “schedule this job to run again.”

### 6. Existing destination surfaces are extensible, but none currently accept source-job seed state

The web destination workspaces already own the right create flows:

- `TemplateManager` supports create, edit, and duplicate from existing template detail
- `WatchContainer` and `WatchManager` already own watch creation and update
- `ExportScheduleContainer` and `ExportScheduleManager` already own schedule creation and update

The current missing piece is not CRUD capability. It is route-to-destination handoff.

None of these surfaces currently accept:

- `sourceJobId`
- `promotionDraft`
- destination-specific seeded draft props
- source-aware empty-state or review UI

This points to a focused cutover: promotion needs route-level orchestration and one-shot seeded draft handoff far more than it needs a brand-new CRUD backend.

### 7. The current browser type contract is safe but weakly typed for promotion mapping

The generated web `Job` type exposes `spec` as a generic object map rather than a strongly typed job-kind union.

That is acceptable for display, but it is a warning sign for promotion implementation. If the cutover relies on browser-side mapping from sanitized job specs into destination draft payloads, that translation should live in one shared mapping layer instead of being rebuilt ad hoc inside `ResultsContainer`, `TemplateManager`, `WatchContainer`, and `ExportScheduleContainer`.

If that mapping becomes too brittle or destination-specific, the server should emit curated promotion drafts instead.

## Product and Contract Decisions

- Keep `SanitizeJob` intact. Promotion must not broaden browser-visible access to secrets or host-local paths.
- Treat `GET /v1/jobs/{id}` as the authoritative route-level source for promotion context; the paged jobs list is not enough.
- Keep template promotion narrow and high-confidence: duplicate an existing named template, or seed from inline extract rules when they already exist.
- Treat watch promotion as scrape-first in the initial cut because the current watch model is single-target monitoring and lacks auth-carrying request fidelity.
- Treat export schedule promotion as future-job export automation, not as replay or rerun scheduling.
- Prefer one central promotion mapping layer. Do not scatter destination-draft derivation logic across multiple web surfaces.
- If client-side mapping from sanitized job data is not robust enough, add narrow server-generated destination draft contracts rather than a raw unsanitized source-job endpoint.

## Recommended Implementation Constraints

### Route and source-of-truth

- Add authoritative `/jobs/:id` detail fetching in the web shell for direct route entry and paged-list misses.
- Keep promotion initiation on `/jobs/:id` and carry only one-shot seeded draft state into the destination workspace.

### Template cutover

- Prefer client-side seeding for named-template duplication and inline-template promotion.
- Do not pretend generic extract or AI settings can always become selector rules without operator review.
- If the source job has no reusable template structure, promote into a guided blank or AI-assisted authoring state instead of failing the flow.

### Watch cutover

- Limit v1 watch promotion to source jobs that can meaningfully seed a single URL watch.
- Keep watch-only inputs explicit: interval, diff style, notification behavior, and change sensitivity.
- Do not imply that auth-backed scrape jobs can become equivalent auth-backed watches until the watch contract actually supports that behavior.
- Decide separately whether optional `jobTrigger` seeding belongs in v1 or should remain a later enhancement.

### Export schedule cutover

- Seed filter intent and export intent, not replay semantics.
- Prefer suggestions such as job kind, successful-status filtering, and has-results defaults.
- Only seed export format automatically when the promotion entry point already has trustworthy export context.
- Keep destination, transform, shape, and retry decisions explicit.

### Security and contract hygiene

- Do not add an unsanitized browser-visible job detail contract.
- If curated destination draft endpoints become necessary, make them destination-specific or return safe normalized draft payloads rather than raw job specs.
- Ensure no promotion surface reintroduces host-local paths, tokens, cookies, or redacted headers through source-context UI.

## Acceptance Criteria

- The implementation team can state, per destination, which source-job fields are safely reusable, which require operator confirmation, and which are unsupported in the first cut.
- The roadmap points to this audit as completed planning input before implementation.
- The main promotion spec reflects the corrected destination semantics from this audit.
- No planned implementation path assumes that the browser can or should receive unsanitized source-job detail.
- The next cut can proceed without re-litigating whether promotion is primarily a backend CRUD problem or a route-and-draft-handoff problem.
