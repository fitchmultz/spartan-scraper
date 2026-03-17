/**
 * Purpose: Render the scan-first jobs monitoring dashboard for the `/jobs` route.
 * Responsibilities: Build lane-based jobs monitoring UI, expose filter/pagination/refresh controls, restore saved jobs-route state, and delegate per-job actions to the parent shell.
 * Scope: `/jobs` route presentation only; data loading and mutations stay in `useAppData()` and `App.tsx`.
 * Usage: Render from the jobs route with authoritative job data and pagination state from the application shell.
 * Invariants/Assumptions: Jobs are already paged/filter-scoped by the parent, failedJobs is a recent-failure subset, and saved jobs-route state should restore once before being cleared.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import type { ManagerStatus } from "../../hooks/useAppData";
import {
  buildJobMonitoringDashboardModel,
  clearJobsViewState,
  getConnectionStateLabel,
  loadJobsViewState,
  type ConnectionState,
  type JobMonitorCardModel,
  type JobStatusFilter,
  type JobsViewState,
} from "../../lib/job-monitoring";
import type { JobEntry } from "../../types";
import { JobCardSkeleton } from "./JobCardSkeleton";
import { JobRunCard } from "./JobRunCard";

const FILTERS: Array<{ label: string; value: JobStatusFilter }> = [
  { label: "All", value: "" },
  { label: "Queued", value: "queued" },
  { label: "Running", value: "running" },
  { label: "Failed", value: "failed" },
  { label: "Succeeded", value: "succeeded" },
  { label: "Canceled", value: "canceled" },
];

interface LaneSectionProps {
  title: string;
  description: string;
  tone: "attention" | "progress" | "completed";
  jobs: JobMonitorCardModel[];
  showSkeletons: boolean;
  emptyMessage: string;
  onViewResults: (jobId: string) => void;
  onCancel: (jobId: string) => void;
  onDelete: (jobId: string) => void;
}

function LaneSection({
  title,
  description,
  tone,
  jobs,
  showSkeletons,
  emptyMessage,
  onViewResults,
  onCancel,
  onDelete,
}: LaneSectionProps) {
  const headingId = `job-lane-${tone}-title`;

  return (
    <section
      className={`job-lane job-lane--${tone}`}
      aria-labelledby={headingId}
    >
      <div className="job-lane__header">
        <div>
          <div className="job-lane__eyebrow">Monitoring lane</div>
          <h3 id={headingId}>{title}</h3>
          <p>{description}</p>
        </div>
        <span className="job-lane__count">{jobs.length}</span>
      </div>

      <div className="job-lane__cards">
        {showSkeletons ? (
          <>
            <JobCardSkeleton />
            <JobCardSkeleton />
          </>
        ) : jobs.length > 0 ? (
          jobs.map((job) => (
            <JobRunCard
              key={job.id}
              model={job}
              lane={tone}
              onViewResults={onViewResults}
              onCancel={onCancel}
              onDelete={onDelete}
            />
          ))
        ) : (
          <div className="job-dashboard-empty">{emptyMessage}</div>
        )}
      </div>
    </section>
  );
}

export interface JobMonitoringDashboardProps {
  jobs: JobEntry[];
  failedJobs: JobEntry[];
  error: string | null;
  loading: boolean;
  statusFilter: JobStatusFilter;
  onStatusFilterChange: (value: JobStatusFilter) => void;
  onViewResults: (jobId: string, format: string, page: number) => void;
  onCancel: (jobId: string) => void;
  onDelete: (jobId: string) => void;
  onRefresh: () => void;
  currentPage: number;
  totalJobs: number;
  jobsPerPage: number;
  onPageChange: (page: number) => void;
  connectionState?: ConnectionState;
  managerStatus?: ManagerStatus | null;
}

export function JobMonitoringDashboard({
  jobs,
  failedJobs,
  error,
  loading,
  statusFilter,
  onStatusFilterChange,
  onViewResults,
  onCancel,
  onDelete,
  onRefresh,
  currentPage,
  totalJobs,
  jobsPerPage,
  onPageChange,
  connectionState = "polling",
  managerStatus,
}: JobMonitoringDashboardProps) {
  const [jumpInputValue, setJumpInputValue] = useState(String(currentPage));
  const [pendingRestore, setPendingRestore] = useState<JobsViewState | null>(
    null,
  );
  const attemptedRestoreRef = useRef(false);

  const maxPage = Math.max(1, Math.ceil(totalJobs / jobsPerPage));

  useEffect(() => {
    setJumpInputValue(String(currentPage));
  }, [currentPage]);

  useEffect(() => {
    if (attemptedRestoreRef.current) {
      return;
    }

    attemptedRestoreRef.current = true;
    const saved = loadJobsViewState();
    if (!saved) {
      return;
    }

    const restoredPage = Math.min(saved.currentPage, maxPage);
    const nextState: JobsViewState = {
      ...saved,
      currentPage: restoredPage,
    };

    setPendingRestore(nextState);

    if (nextState.statusFilter !== statusFilter) {
      onStatusFilterChange(nextState.statusFilter);
    }

    if (nextState.currentPage !== currentPage) {
      onPageChange(nextState.currentPage);
    }
  }, [currentPage, maxPage, onPageChange, onStatusFilterChange, statusFilter]);

  useEffect(() => {
    if (!pendingRestore || typeof window === "undefined") {
      return;
    }

    if (pendingRestore.statusFilter !== statusFilter) {
      return;
    }

    if (pendingRestore.currentPage !== currentPage) {
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      window.scrollTo({
        top: pendingRestore.scrollY,
        behavior: "auto",
      });
      clearJobsViewState();
      setPendingRestore(null);
    });

    return () => window.cancelAnimationFrame(frame);
  }, [currentPage, pendingRestore, statusFilter]);

  const dashboard = useMemo(
    () =>
      buildJobMonitoringDashboardModel({
        jobs,
        failedJobs,
        totalJobs,
        connectionState,
        managerStatus,
      }),
    [jobs, failedJobs, totalJobs, connectionState, managerStatus],
  );

  const isRestoringFilters =
    pendingRestore !== null &&
    (pendingRestore.statusFilter !== statusFilter ||
      pendingRestore.currentPage !== currentPage);
  const showSkeletons = (loading && jobs.length === 0) || isRestoringFilters;

  const handleViewResults = (jobId: string) => {
    onViewResults(jobId, "jsonl", 1);
  };

  const connectionToneClass =
    connectionState === "connected"
      ? "job-summary-card--healthy"
      : connectionState === "reconnecting" || connectionState === "polling"
        ? "job-summary-card--active"
        : "job-summary-card--attention";

  return (
    <section
      className="panel job-monitoring-dashboard"
      data-tour="jobs-dashboard"
    >
      <div className="job-summary-strip">
        <div className="job-summary-card">
          <span className="job-summary-card__label">Total jobs</span>
          <strong className="job-summary-card__value">
            {dashboard.summary.totalJobs}
          </strong>
        </div>

        <div className="job-summary-card job-summary-card--active">
          <span className="job-summary-card__label">Queued</span>
          <strong className="job-summary-card__value">
            {dashboard.summary.queued}
          </strong>
        </div>

        <div className="job-summary-card job-summary-card--active">
          <span className="job-summary-card__label">Running</span>
          <strong className="job-summary-card__value">
            {dashboard.summary.running}
          </strong>
        </div>

        <div className="job-summary-card job-summary-card--attention">
          <span className="job-summary-card__label">Recent failures</span>
          <strong className="job-summary-card__value">
            {dashboard.summary.recentFailures}
          </strong>
        </div>

        <div className={`job-summary-card ${connectionToneClass}`}>
          <span className="job-summary-card__label">Connection</span>
          <strong className="job-summary-card__value">
            {getConnectionStateLabel(dashboard.summary.connectionState)}
          </strong>
        </div>
      </div>

      <div className="job-dashboard-controls">
        <div
          className="job-dashboard-filters"
          role="toolbar"
          aria-label="Job status filters"
        >
          {FILTERS.map((filter) => (
            <button
              key={filter.label}
              type="button"
              className={
                statusFilter === filter.value
                  ? "job-filter-chip is-active"
                  : "job-filter-chip"
              }
              aria-pressed={statusFilter === filter.value}
              onClick={() => onStatusFilterChange(filter.value)}
            >
              {filter.label}
            </button>
          ))}
        </div>

        <div className="job-dashboard-controls__actions">
          <div className="pagination-controls job-dashboard-pagination">
            <button
              type="button"
              disabled={currentPage <= 1}
              onClick={() => onPageChange(currentPage - 1)}
            >
              Previous
            </button>

            <span className="pagination-info">
              Page {currentPage} of {maxPage} ({totalJobs} jobs)
            </span>

            <button
              type="button"
              disabled={currentPage >= maxPage}
              onClick={() => onPageChange(currentPage + 1)}
            >
              Next
            </button>

            <div className="pagination-jump">
              <input
                type="number"
                min="1"
                max={maxPage}
                value={jumpInputValue}
                onChange={(event) => setJumpInputValue(event.target.value)}
              />
              <button
                type="button"
                onClick={() => {
                  const page = Number.parseInt(jumpInputValue, 10);
                  if (Number.isInteger(page) && page >= 1 && page <= maxPage) {
                    onPageChange(page);
                  }
                }}
              >
                Go
              </button>
            </div>
          </div>

          <button type="button" className="secondary" onClick={onRefresh}>
            Refresh
          </button>
        </div>
      </div>

      {error ? (
        <div className="error" role="alert">
          {error}
        </div>
      ) : null}

      <div className="job-lane-grid">
        <LaneSection
          title="Needs Attention"
          description="Failed runs and dependency-blocked work that need operator intervention."
          tone="attention"
          jobs={dashboard.lanes.attention}
          showSkeletons={showSkeletons}
          emptyMessage="No failures or blocked jobs need attention right now."
          onViewResults={handleViewResults}
          onCancel={onCancel}
          onDelete={onDelete}
        />

        <LaneSection
          title="In Progress"
          description="Queued and running work, with progress and timing surfaced first."
          tone="progress"
          jobs={dashboard.lanes.progress}
          showSkeletons={showSkeletons}
          emptyMessage="No queued or running jobs in this view."
          onViewResults={handleViewResults}
          onCancel={onCancel}
          onDelete={onDelete}
        />

        <LaneSection
          title="Recent Completed"
          description="Successful and canceled jobs with lower visual emphasis."
          tone="completed"
          jobs={dashboard.lanes.completed}
          showSkeletons={showSkeletons}
          emptyMessage="No recent completed jobs for this page and filter."
          onViewResults={handleViewResults}
          onCancel={onCancel}
          onDelete={onDelete}
        />
      </div>
    </section>
  );
}
