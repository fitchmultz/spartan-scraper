/**
 * Purpose: Render a single scan-friendly job card inside the jobs monitoring dashboard.
 * Responsibilities: Present badges, identifiers, progress, timeline, failure context, and operator actions for one job.
 * Scope: Jobs dashboard card presentation only.
 * Usage: Render from `JobMonitoringDashboard` with a `JobMonitorCardModel` derived by `job-monitoring.ts`.
 * Invariants/Assumptions: Action availability is precomputed in the view model, and lane tone comes from the parent dashboard grouping.
 */

import { getJobStatusBadgeClass, getJobStatusIcon } from "../../lib/job-status";
import type { JobMonitorCardModel } from "../../lib/job-monitoring";
import { JobFailureRail } from "./JobFailureRail";
import { JobProgressBar } from "./JobProgressBar";

interface JobRunCardProps {
  model: JobMonitorCardModel;
  lane: "attention" | "progress" | "completed";
  onViewResults: (jobId: string) => void;
  onCancel: (jobId: string) => void;
  onDelete: (jobId: string) => void;
}

export function JobRunCard({
  model,
  lane,
  onViewResults,
  onCancel,
  onDelete,
}: JobRunCardProps) {
  return (
    <article className={`job-run-card job-run-card--${lane}`}>
      <header className="job-run-card__header">
        <div className="job-run-card__badges">
          <span className="job-run-card__kind">{model.kind}</span>
          <span className={`badge ${getJobStatusBadgeClass(model.status)}`}>
            {model.status}
          </span>

          {model.dependencyStatus && model.dependencyStatus !== "ready" ? (
            <span
              className={`badge ${getJobStatusBadgeClass(model.dependencyStatus)}`}
              title={
                model.dependencyStatus === "pending"
                  ? "Waiting for dependencies"
                  : "Dependency failed"
              }
            >
              deps: {model.dependencyStatus}
            </span>
          ) : null}

          {model.chainId ? (
            <span className="badge" title={`Chain ${model.chainId}`}>
              chain {model.chainId.slice(0, 6)}
            </span>
          ) : null}
        </div>

        <span className="job-run-card__updated">{model.updatedAtLabel}</span>
      </header>

      <div className="job-run-card__title-block">
        <h4>
          <span aria-hidden="true">{getJobStatusIcon(model.status)}</span>{" "}
          {model.kind} · {model.shortId}
        </h4>
        <p className="job-run-card__raw-id">{model.rawId}</p>
      </div>

      <div className="job-run-card__details">
        <p className="job-run-card__activity">{model.activityText}</p>
        {model.dependsOnLabel ? (
          <p className="job-run-card__depends">{model.dependsOnLabel}</p>
        ) : null}
      </div>

      <JobProgressBar progress={model.progress} />

      <dl className="job-run-card__timeline">
        {model.timeline.map((item) => (
          <div key={item.label}>
            <dt>{item.label}</dt>
            <dd>{item.value}</dd>
          </div>
        ))}
      </dl>

      <JobFailureRail failure={model.failure} />

      <div className="job-run-card__actions">
        {model.canViewResults ? (
          <button
            type="button"
            className="secondary"
            onClick={() => onViewResults(model.id)}
          >
            View Results
          </button>
        ) : null}

        {model.canCancel ? (
          <button
            type="button"
            className="secondary"
            onClick={() => onCancel(model.id)}
          >
            Cancel
          </button>
        ) : null}

        {model.canDelete ? (
          <button
            type="button"
            className="secondary job-run-card__delete"
            onClick={() => onDelete(model.id)}
          >
            Delete
          </button>
        ) : null}
      </div>
    </article>
  );
}
