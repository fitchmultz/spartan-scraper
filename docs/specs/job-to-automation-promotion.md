# Verified Job Promotion Flow

**Status:** Planned  
**Primary surfaces:** Web UI `/jobs/:id`, `/templates`, `/automation/watches`, `/automation/exports`

## Summary

Allow operators to turn a completed, trusted job into a reusable template, watch, or export schedule without re-entering known-good configuration from scratch.

The product already helps operators submit work, inspect results, and manage automation, but there is no operator-grade bridge between those surfaces. The right flow should begin where trust is established — the completed job detail and results experience — then hand off into the canonical destination workspace for final review and save.

This keeps the route model intact while removing the manual re-entry tax that currently sits between “this worked” and “make this reusable.”

## Problems This Solves

- Operators must manually re-enter configuration that already exists in a completed job.
- The results surface can suggest next steps, but those suggestions do not currently lead to real promotion actions.
- `/templates`, `/automation/watches`, and `/automation/exports` all begin from blank or scratch-oriented creation flows even when the source job is already known-good.
- Existing product copy in Settings and inventory panels talks about promoting verified work into automation, but that promise is not yet backed by real UX.
- The current gap breaks momentum at the exact point where operators should be able to turn manual success into repeatable value.

## Product Decisions

- `/jobs/:id` is the primary origin for promotion because it is where operators confirm whether a job is worth reusing.
- Promotion should be operator-initiated from a succeeded job in this cut. If the product later introduces an explicit verified or approved state, it should extend the same flow rather than create a separate one.
- Promotion should use one shared in-context chooser rather than multiple disconnected creation buttons scattered across the page.
- Promotion should remain destination-specific. Operators choose whether they want a template, a watch, or an export schedule instead of one action that creates all three at once.
- The destination routes remain canonical ownership surfaces:
  - `/templates` for template authoring
  - `/automation/watches` for watch creation and maintenance
  - `/automation/exports` for export schedule creation and maintenance
- The promotion flow should navigate into those destination workspaces with a seeded draft rather than silently creating an artifact without review.
- Prefill should draw first from safely recoverable operator-facing job data, then from destination-relevant outcome context when available. Redacted secrets, paths, and unsupported destination fields should remain explicit operator decisions rather than hidden defaults.
- Watch promotion in the first cut should be limited to source jobs that can meaningfully seed a single-target monitoring flow, with scrape-job sources as the default supported path.
- Export schedule promotion must be framed as automated export of future matching completed jobs, not as rerunning the source job on a cadence.
- Missing destination-specific decisions should stay visible and explicit. The system should not hide uncertainty behind silent defaults or a fully blank form.
- Before implementation, product and API work should confirm what source-job-based draft or cloning support already exists and only add new contracts where necessary to preserve a clean promotion experience.

## Goals

- Let operators move from manual success to reusable automation without duplicate data entry.
- Make the choice between template, watch, and export schedule understandable from the context of a successful job.
- Preserve route clarity by starting promotion on the job surface and finishing it inside the destination workspace.
- Reuse as much trustworthy source-job context as possible.
- Keep operator confidence high by requiring destination review before save.
- Turn recommended actions and existing “promote this flow” product language into real behavior.

## Non-Goals

- Creating a new top-level promotion route.
- Automatically creating multiple automation artifacts from one click.
- Introducing a new persisted “verified” job model as part of this cut.
- Reworking CLI, MCP, or TUI promotion flows in the same effort.
- Redesigning template, watch, or export management more broadly than required to support a sourced draft.
- Solving every future automation handoff problem beyond template, watch, and export schedule promotion.

## Interaction Model

### Origin surface

Promotion should begin inside the completed job experience, not from Settings and not from blank destination routes.

The primary promotion entry belongs on `/jobs/:id` in a visible action position appropriate for a completed job. Secondary contextual entry points can also appear inside result-specific recommendation surfaces, such as suggested next steps connected to export outcomes. Those entry points must converge on the same promotion model, not branch into separate UX paths.

The promotion interaction itself should stay in-context to the job route. It should help the operator choose the right destination without becoming a new route of its own. Once the operator chooses a destination, the product should navigate to that destination’s canonical workspace with the source job attached as the seed.

### Promotion chooser

The chooser should present three destination options with plain-language guidance:

- **Save as Template** — best when the operator wants to preserve a successful extraction or request recipe for reuse and adaptation.
- **Create Watch** — best when the operator wants Spartan to monitor a single verified target over time for change.
- **Create Export Schedule** — best when the operator wants future matching completed jobs to export automatically without repeating manual export setup.

Each option should explain two things clearly:

1. what Spartan can carry forward from the completed job
2. what the operator will still need to confirm in the destination workspace

The chooser should help operators make the right decision, not just expose raw system nouns.

### Eligibility

Promotion should be available for succeeded jobs in this cut. Failed, canceled, queued, and running jobs should not present the same promotion path as if they were valid seeds.

If a future product state adds explicit verification or approval, that state should refine the same flow rather than require a new information architecture.

### Source context

Every promoted draft should preserve visible source context so the operator understands where it came from. That context should include the source job identity and the part of the original request or run that matters to the chosen destination, such as target URL or query, completion timing, and a clear way to return to the source job.

Promotion should feel like “create from this known-good job,” not “open a mostly blank editor that happens to remember a hidden ID.”

### Destination requirements

#### Template promotion

Template creation should carry forward the parts of the source job that define reusable extraction behavior.

That can include:

- the source target or representative input
- extraction instructions, rules, or template-relevant configuration already present as reusable template structure
- non-secret preview context that helps the template workspace stay grounded in a real successful run
- source-job context that helps preview and debugging workflows stay anchored to the successful job without silently turning runtime-only settings into template payload

The destination should suggest a sensible name and metadata based on the source job, but still require operator review before save.

One-off run artifacts or transient execution details should not be silently baked into a reusable template unless they are clearly being promoted as reusable settings.

#### Watch promotion

Watch creation should carry forward the parts of the source job that actually map to the current watch contract: the target URL, browser/runtime choices such as headless or Playwright when relevant, and any screenshot-related settings that help monitor the same target.

The product should also make the relationship to the source job obvious so the operator understands what known-good baseline they are starting from.

The current watch contract is a single-target monitoring configuration, not a full reusable job clone. It does not currently carry general auth context, extraction templates, or broader pipeline configuration. Promotion from a completed job should therefore stay explicit about what was preserved, what still needs operator input, and what is unsupported in the first cut.

#### Export schedule promotion

Export schedule creation should begin from the source job context and any destination-relevant export choices that already exist from the completed run.

Export schedules in Spartan automate export for future matching completed jobs; they do not rerun the source job on a cadence. The promoted draft should therefore focus on sensible filters, export intent, and delivery defaults rather than pretending it can recreate the entire manual workflow.

If the operator already chose an export format or similar result-delivery setup from the job context, the destination should preserve that where it makes sense. If that information does not exist, the product should still launch a real schedule draft from the source job and clearly highlight the missing export-specific decisions instead of dumping the operator into a blank create flow.

Matching filters, destination settings, and delivery-specific choices still require explicit operator confirmation.

### Handoff and review

The destination workspace should open in a clear draft or create-from-job state. It should not immediately persist a new artifact without review.

The operator should see:

- that this draft came from a specific completed job
- which parts of the configuration were carried forward
- which required decisions remain before save
- a straightforward way to return to the source job if more inspection is needed

This preserves trust and prevents hidden cloning behavior.

### Recommended action alignment

Any result-side recommendation that implies “turn this into automation” must lead to a working destination. Text-only advice is not enough once the product claims the path exists.

Where the recommendation context is already specific, the product can deep-link directly into the most natural destination draft. The main promotion action on the job detail surface should still remain available for operators who want to choose among all three destinations.

### Contract audit findings

See [Verified Job Promotion Contract Audit](job-to-automation-promotion-contract-audit.md) for the confirmed source-of-truth and gap analysis that now governs implementation.

The audit locked in these constraints:

- `GET /v1/jobs/{id}` already exists and should become the authoritative route-level source when `/jobs/:id` is opened for a job outside the current paged jobs list.
- The Web UI should continue to treat `SanitizeJob` output as the trusted browser contract. Promotion must not depend on exposing unsanitized job detail or raw secrets in the browser.
- Template promotion is high-confidence only when the source job already contains reusable extraction structure, such as a named template or inline template rules. Other extraction modes should fall back to a guided blank or AI-assisted template draft.
- Watch promotion is not a full job clone. The current watch contract is a single-target monitoring model with no general auth or pipeline support, so the first cut should be scrape-first and explicit about unsupported carry-forward.
- Export schedule promotion must represent future-job export automation, not repeated execution of the source job.
- If sanitized source-job data plus existing destination lookups prove insufficient, prefer narrow server-generated draft endpoints over any unsanitized raw job-detail contract.

## Acceptance Criteria

- From a succeeded `/jobs/:id` view, an operator can initiate promotion to a template, watch, or export schedule without manually navigating blind into a blank create flow.
- Result-side recommendations that imply reusable automation land in a real promotion path instead of dead-ending as advisory text.
- `/templates`, `/automation/watches`, and `/automation/exports` can each open in a source-job-seeded create state with visible source context.
- Promotion still works when `/jobs/:id` is opened directly for a job that is not already present in the current paged jobs list.
- Supported reusable fields are prefilled from the source job where appropriate for the chosen destination.
- Destination-specific required fields that cannot be inferred from the source job are clearly identified instead of silently defaulted or left unexplained.
- Promotion preserves the existing redaction boundary and never requires unsanitized browser-visible job detail.
- Promotion never silently persists an artifact without operator review.
- Existing product language about promoting successful manual work into automation is backed by real product behavior once this spec is delivered.
