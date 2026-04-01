/**
 * Purpose: Provide reusable job status helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
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
