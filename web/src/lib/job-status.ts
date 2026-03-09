/**
 * Shared job status presentation helpers.
 *
 * Purpose:
 * - Centralize icon and badge-class mapping for job and dependency statuses.
 *
 * Responsibilities:
 * - Provide one typed source of truth for job status icons.
 * - Provide consistent badge styling for job and dependency state displays.
 *
 * Scope:
 * - Web UI presentation helpers for job-related status values only.
 *
 * Usage:
 * - Import from components that render job badges, dependency badges, or job
 *   status labels.
 *
 * Invariants/Assumptions:
 * - Unknown statuses fall back to neutral display values.
 * - Queued and pending states should render as in-progress badges.
 */

import type { JobEntry } from "../types";

type JobStatus = JobEntry["status"];
type JobDependencyStatus = NonNullable<JobEntry["dependencyStatus"]>;
type JobBadgeStatus = JobStatus | JobDependencyStatus;

export function getJobStatusBadgeClass(status?: JobBadgeStatus): string {
  switch (status) {
    case "queued":
    case "running":
    case "pending":
      return "running";
    case "succeeded":
    case "ready":
      return "success";
    case "failed":
    case "canceled":
      return "failed";
    default:
      return "";
  }
}

export function getJobStatusIcon(status?: JobStatus): string {
  switch (status) {
    case "running":
      return "▶️";
    case "succeeded":
      return "✅";
    case "failed":
      return "❌";
    case "canceled":
      return "⏹️";
    case "queued":
      return "⏳";
    default:
      return "📄";
  }
}
