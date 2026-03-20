# Promotion Flow Deterministic Regression Coverage

**Status:** Planned  
**Primary surfaces:** `internal/system/`, targeted browser coverage for `/jobs/:id`, `/templates`, `/automation/watches`, and `/automation/exports`

## Summary

Define the regression strategy for the verified job promotion flow so it cannot silently break as job requests, destination editors, or automation contracts evolve.

The primary guarantee should live in deterministic system tests that run in `make test-ci`. Browser-level coverage should exist, but only as a narrow secondary layer that proves critical operator affordances and route handoff wiring. The goal is to protect the operator journey from known-good job to reusable automation without making the core regression story slow or flaky.

## Problems This Solves

- Promotion spans multiple product surfaces and backend domains: jobs, templates, watches, export schedules, and route handoff.
- Source-job prefill can silently degrade when request models or destination editors change.
- Results recommendations and navigation affordances can regress even when underlying backend behavior still works.
- Browser-only coverage would make the primary regression story slower and less deterministic than necessary.
- Without an explicit definition of what regression means and where it should be tested, coverage is likely to stop at one happy path or be postponed entirely.

## Product Decisions

- Deterministic system tests in `internal/system/` are the primary owner of promotion regression coverage.
- Primary promotion regression must run without Playwright and belong in the normal `make test-ci` path.
- Browser coverage is secondary and should stay intentionally small. Its job is to prove operator-visible affordances and route handoff, not to duplicate all system coverage.
- Regression means more than “the request returned 200.” It includes:
  - promotion eligibility
  - route or contract handoff from source job to destination
  - preservation of reusable source-job context
  - explicit handling of destination-only missing fields
  - successful creation of the intended reusable artifact without mutating the source job
- Template, watch, and export schedule promotion each need explicit coverage. One shared helper-level assertion is not enough.
- Negative cases matter. Ineligible jobs and incomplete source context should be tested on purpose.
- Dogfooding remains valuable acceptance input, but it is not a substitute for deterministic regression coverage.

## Goals

- Catch promotion contract drift in the normal local CI path.
- Keep the primary regression story deterministic and browser-free.
- Validate the end-to-end operator-critical promotion flows for templates, watches, and export schedules.
- Define a clean split between what system tests own and what browser tests own.
- Set a clear threshold for when promotion regression coverage is enough.

## Non-Goals

- Exhaustive visual regression coverage for the promotion UI.
- Full browser duplication of the system-level promotion matrix.
- Live-network testing.
- Broad expansion into unrelated route coverage.
- Testing every field permutation or every future automation type.

## Coverage Model

### What counts as a promotion regression

A promotion regression is any change that causes one of the following:

- a succeeded job can no longer be used as a valid promotion source when it should be
- an ineligible job is incorrectly treated as promotable
- a promotion entry or handoff no longer reaches the intended destination
- reusable source-job request or outcome context is lost during handoff
- a created draft or artifact no longer contains the expected reusable configuration
- destination-specific missing information stops being explicit
- the source job is mutated as a side effect of promotion
- a created template, watch, or export schedule no longer behaves like a real promoted artifact and instead degrades into a blank or detached object

### System-level coverage

System tests should be the main line of defense because the hardest problems here are contract stability, request reuse, artifact creation, and end-to-end orchestration.

The primary system coverage should include the following categories.

#### 1. Eligibility and guardrails

Verify that:

- succeeded jobs can be used as promotion sources
- queued, running, failed, or canceled jobs are rejected when the product says they are ineligible
- rejected promotion attempts preserve clear error semantics
- source jobs remain unchanged after promotion attempts

#### 2. Template promotion

Cover a representative completed job and assert that template promotion preserves the reusable parts of the original work.

That includes validation that:

- the promotion path can create or seed a template from a completed job
- the resulting template is not blank
- the promoted template preserves the expected reusable request or extraction context
- the promoted template retains source-job lineage or reference where the product contract says it should

#### 3. Watch promotion

Cover a representative completed job and assert that watch promotion preserves the reusable monitoring seed.

That includes validation that:

- the promotion path can create or seed a watch from a completed job
- the resulting watch carries forward target and reusable request context
- watch-specific decisions that are not inferable from the source job stay explicit rather than being silently guessed
- the resulting watch is a usable promoted artifact, not a detached placeholder

#### 4. Export schedule promotion

Cover a representative completed job and assert that export schedule promotion preserves the reusable schedule seed.

That includes validation that:

- the promotion path can create or seed an export schedule from a completed job
- export-relevant source context is preserved when available
- schedule-specific decisions that require operator confirmation remain explicit when the source job does not determine them
- the resulting schedule is a usable promoted artifact

#### 5. Operator journey coverage

Extend the existing operator-flow style coverage so at least one deterministic end-to-end system scenario exercises:

1. creating or running a manual job
2. reaching successful completion
3. promoting that completed job into a reusable artifact
4. verifying the destination artifact exists and is accessible in its management surface

This proves the promotion flow works as part of the real operator journey, not just as an isolated API call.

#### 6. Recommendation and handoff contract coverage

If the platform exposes recommendation or next-step metadata for the results surface, the regression suite should verify that promotion-oriented recommendations map to a valid destination and do not degrade into dead-end advisory text.

### Browser-level coverage

Browser coverage is still useful, but it should remain intentionally narrow.

The browser layer should prove:

- a succeeded job visibly exposes the main promotion affordance on `/jobs/:id`
- invoking that affordance reaches the correct destination route
- the destination visibly reflects source-job context and representative prefilled state
- if the results surface exposes contextual automation recommendations, at least one such recommendation reaches the intended destination

This browser layer should live in the slower path and should not become the primary truth source for promotion correctness.

### Enough threshold

Promotion regression coverage is enough when all of the following are true:

- `make test-ci` includes deterministic promotion coverage for template, watch, and export schedule flows
- `make test-ci` includes at least one ineligible-source or guardrail case
- at least one operator-journey system test proves manual success can become reusable automation
- browser coverage exists for the visible job-detail promotion entry and at least one source-job-to-destination handoff
- regressions in source-job reuse, destination seeding, or promotion eligibility produce deterministic failures instead of depending on manual discovery

## Acceptance Criteria

- The promotion flow has deterministic system regression coverage for template, watch, and export schedule promotion.
- The primary promotion regression suite runs without Playwright and belongs in `make test-ci`.
- Negative coverage exists for at least one ineligible source-job scenario.
- Browser coverage, if added, stays focused on visible affordances and destination handoff rather than duplicating the full system matrix.
- Promotion assertions verify preserved source context and destination seeding explicitly, not just status codes or object existence.
- Future changes that break source-job-to-automation promotion fail deterministically in regression coverage instead of being caught only through ad hoc dogfooding.
