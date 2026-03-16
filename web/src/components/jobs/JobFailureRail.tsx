/**
 * Purpose: Surface failure severity and summary context for job monitoring cards.
 * Responsibilities: Render a compact, tone-aware failure callout with accessible labeling for triage.
 * Scope: Jobs dashboard failure presentation only.
 * Usage: Render inside `JobRunCard` with the failure model produced by `job-monitoring.ts`.
 * Invariants/Assumptions: Missing failure data means no failure rail is shown, and tone values map directly to CSS modifiers.
 */

import type { JobMonitorCardModel } from "../../lib/job-monitoring";

interface JobFailureRailProps {
  failure?: JobMonitorCardModel["failure"];
}

export function JobFailureRail({ failure }: JobFailureRailProps) {
  if (!failure) {
    return null;
  }

  return (
    <div
      className={`job-failure-rail job-failure-rail--${failure.tone}`}
      role="note"
      aria-label={`${failure.category} failure context`}
    >
      <div className="job-failure-rail__eyebrow">{failure.category}</div>
      <p>{failure.summary}</p>
    </div>
  );
}
