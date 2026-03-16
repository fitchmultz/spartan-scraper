# Job Monitoring Dashboard

**Status:** Roadmap / Now  
**Primary surface:** Web UI `/jobs` and related return flow from `/jobs/:id`

## Summary

Rework the jobs monitoring experience from a dense, log-like list into a scan-first dashboard for operators who need to understand queue state, active work, failures, and recent completions quickly.

The current `JobList` is functional, but it is visually flat, overuses monospace presentation, and hides the most important operational signals inside text blobs. This spec defines a product-grade monitoring surface without requiring a backend rewrite.

## Problems This Solves

- Dense job rows are hard to scan at a glance.
- Running work has no strong visual progress treatment.
- Failed jobs are surfaced, but not in a way that supports rapid triage.
- Batch progression is text-only even when queue data is available.
- Moving from `/jobs` to `/jobs/:id` breaks context because filters, paging, and scroll position are not preserved.

## Product Decisions

- Keep `/jobs` as the top-level monitoring route.
- Keep `/jobs/:id` as the result-focused detail route.
- Restore prior jobs-route context when returning from job detail.
- Do not require new API contracts for the first cut. Use the existing `JobEntry` shape and derive display models in the Web UI.
- If richer queue or failure metadata is already available, use it, but do not block implementation on new backend work.

## Goals

- Make active, blocked, and failed work visually obvious within seconds.
- Add progress bars for running and batched work.
- Improve failure visibility with clearer severity styling and actionable next steps.
- Preserve existing mutations: refresh, cancel, delete, and view results.
- Replace generic loading text with skeleton states.
- Remove inline-style-heavy presentation in favor of class-based styling using theme tokens.

## Non-Goals

- Replacing the jobs API.
- Adding server-side search or new filtering contracts.
- Reworking CLI, MCP, or TUI monitoring surfaces in this cut.
- Inventing retry semantics if no backend retry capability exists.

## Primary User Workflows

1. **Scan active queue**
   - See queued and running work first.
   - Understand whether a job is blocked on dependencies.
   - Read batch progress without parsing text.

2. **Triage failures**
   - See recent failures grouped as a high-attention lane.
   - Read failure category and summary quickly.
   - Take immediate actions: inspect context, filter, delete, or navigate as supported.

3. **Monitor completion**
   - See recent successful work in a lower-priority completed lane.
   - Jump to results from a clear action target.

4. **Open job detail and return without losing place**
   - Open `/jobs/:id`.
   - Return to the same page, filter, and approximate scroll position on `/jobs`.

## Information Architecture

### `/jobs` route layout

1. **Summary strip**
   - Total jobs
   - Queued
   - Running
   - Recent failures
   - Connection state

2. **Control row**
   - Status filter chips
   - Refresh action
   - Pagination controls
   - Optional density toggle only if it does not add meaningful complexity

3. **Needs Attention lane**
   - Recent failed jobs
   - Dependency-blocked jobs
   - Strong failure styling and quick actions

4. **In Progress lane**
   - Running and queued jobs
   - Visual progress indicators
   - Queue position visualization
   - Duration/timeline summary

5. **Recent Completed lane**
   - Successful and canceled jobs
   - Lower visual emphasis than failures and active work

### `/jobs/:id` return behavior

When the user navigates from the jobs route into job detail, persist and restore:

- `statusFilter`
- `currentPage`
- scroll position
- optionally selected lane or anchor target if introduced

## Card Anatomy

Each job card should include:

- **Header**
  - job kind
  - status badge
  - dependency badge if present
  - chain badge if present
  - updated time / relative recency

- **Primary title block**
  - human-readable identifier treatment
  - raw ID available, but not visually dominant

- **Progress area**
  - batch progress bar when `job.run.queue` exists
  - indeterminate running bar when job is running without percent data
  - queue position text near the bar, not buried below other metadata

- **Timeline area**
  - wait time
  - run time
  - total time
  - recent activity text such as “waiting for dependencies”, “running fetch”, or “completed”

- **Failure area**
  - failure category
  - failure summary
  - severity styling
  - expandable detail space if more failure context exists

- **Action row**
  - View Results for succeeded jobs
  - Cancel for queued/running jobs
  - Delete for terminal jobs
  - one additional non-destructive utility action if it is cheap to add

## Derived UI Model

Use a small view-model layer instead of rendering raw `JobEntry` directly.

```ts
export interface JobMonitorCardModel {
  id: string;
  shortId: string;
  kind: string;
  status: "queued" | "running" | "succeeded" | "failed" | "canceled";
  dependencyStatus?: "ready" | "pending" | "failed";
  chainId?: string;
  updatedAtLabel: string;
  progress?: {
    label: string;
    percent?: number;
    valueText: string;
    indeterminate?: boolean;
  };
  timeline: Array<{
    label: string;
    value: string;
  }>;
  failure?: {
    tone: "danger" | "warning";
    category: string;
    summary: string;
  };
}
```

## Context Preservation

Persist jobs view state before navigating away.

```ts
type JobsViewState = {
  statusFilter: "" | "queued" | "running" | "succeeded" | "failed" | "canceled";
  currentPage: number;
  scrollY: number;
};

const JOBS_VIEW_STATE_KEY = "spartan.jobs.view-state";
```

## Suggested Component Plan

- `web/src/components/jobs/JobMonitoringDashboard.tsx`
- `web/src/components/jobs/JobRunCard.tsx`
- `web/src/components/jobs/JobProgressBar.tsx`
- `web/src/components/jobs/JobFailureRail.tsx`
- `web/src/components/jobs/JobCardSkeleton.tsx`

If renaming is too disruptive, keep `JobList.tsx` as the exported boundary and move new subcomponents under `components/jobs/`.

## Acceptance Criteria

- The jobs route can be scanned quickly for failures, active work, and completion state.
- Running and batched work has visible progress treatment.
- Failed jobs feel actionable, not merely listed.
- The jobs route no longer relies on dense inline-styled monospace rows.
- Navigating to a job detail and back restores prior jobs-route context.
- Loading states use skeletons instead of generic text.
- The new UI works in both dark and light themes using CSS variables.
