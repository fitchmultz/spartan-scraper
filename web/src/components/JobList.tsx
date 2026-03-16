/**
 * Purpose: Preserve the historic `JobList` import boundary while exporting the redesigned jobs monitoring dashboard.
 * Responsibilities: Re-export the new jobs dashboard component and props under the legacy module path.
 * Scope: Compatibility boundary for internal web UI imports only.
 * Usage: Import `JobList` from this path until all callers are ready to reference `JobMonitoringDashboard` directly.
 * Invariants/Assumptions: The application shell still imports `JobList`, and the implementation now lives under `components/jobs/`.
 */

export {
  JobMonitoringDashboard as JobList,
  type JobMonitoringDashboardProps as JobListProps,
} from "./jobs/JobMonitoringDashboard";
