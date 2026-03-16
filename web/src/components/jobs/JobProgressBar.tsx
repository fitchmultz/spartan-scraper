/**
 * Purpose: Render determinate and indeterminate progress treatments for job monitoring cards.
 * Responsibilities: Present queue progress labels, accessible progress semantics, and the visual bar treatment for running or batched work.
 * Scope: Jobs dashboard progress UI only.
 * Usage: Render inside `JobRunCard` with the progress object produced by `job-monitoring.ts`.
 * Invariants/Assumptions: Missing progress means no progress UI should render, and accessible labels/text come from the supplied view model.
 */

import type { JobMonitorCardModel } from "../../lib/job-monitoring";

interface JobProgressBarProps {
  progress?: JobMonitorCardModel["progress"];
}

export function JobProgressBar({ progress }: JobProgressBarProps) {
  if (!progress) {
    return null;
  }

  const isIndeterminate =
    progress.indeterminate || typeof progress.percent !== "number";

  return (
    <div className="job-progress">
      <div className="job-progress__meta">
        <strong>{progress.label}</strong>
        <span>{progress.valueText}</span>
      </div>

      {isIndeterminate ? (
        <div
          className="job-progress__track is-indeterminate"
          role="progressbar"
          aria-label={progress.label}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuetext={progress.valueText}
        >
          <span className="job-progress__indeterminate" />
        </div>
      ) : (
        <progress
          className="job-progress__bar"
          aria-label={progress.label}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={progress.percent}
          aria-valuetext={progress.valueText}
          max={100}
          value={progress.percent}
        >
          {progress.percent}
        </progress>
      )}
    </div>
  );
}
