/**
 * Purpose: Render recent-run inspection for the web UI.
 * Responsibilities: Show recent runs with pagination, status filters, queue/failure context, recent failed-run callouts, and action buttons; surface transport connection state and manual refresh controls.
 * Scope: Presentation only; data fetching and mutation handlers are supplied by the parent container.
 * Usage: Render from the jobs route or job-detail sidebar with authoritative job envelopes from `useAppData()`.
 * Invariants/Assumptions: Jobs already include derived `run` fields, failedJobs is a small recent-failure subset, and pagination is controlled by the parent component.
 */
import { useEffect, useState } from "react";
import { getJobStatusBadgeClass } from "../lib/job-status";
import type { JobEntry } from "../types";

interface JobListProps {
  jobs: JobEntry[];
  failedJobs: JobEntry[];
  error: string | null;
  statusFilter: "" | "queued" | "running" | "succeeded" | "failed" | "canceled";
  onStatusFilterChange: (value: JobListProps["statusFilter"]) => void;
  onViewResults: (jobId: string, format: string, page: number) => void;
  onCancel: (jobId: string) => void;
  onDelete: (jobId: string) => void;
  onRefresh: () => void;
  currentPage: number;
  totalJobs: number;
  jobsPerPage: number;
  onPageChange: (page: number) => void;
  connectionState?: "connected" | "disconnected" | "reconnecting" | "polling";
}

function ConnectionIndicator({
  state,
}: {
  state: "connected" | "disconnected" | "reconnecting" | "polling";
}) {
  const indicatorStyle: React.CSSProperties = {
    width: 8,
    height: 8,
    borderRadius: "50%",
    display: "inline-block",
    marginRight: 6,
  };

  const containerStyle: React.CSSProperties = {
    display: "flex",
    alignItems: "center",
    fontSize: 12,
    color: "#666",
    padding: "4px 8px",
    backgroundColor: "#f5f5f5",
    borderRadius: 4,
  };

  switch (state) {
    case "connected":
      return (
        <span
          style={containerStyle}
          title="WebSocket connected - real-time updates"
        >
          <span style={{ ...indicatorStyle, backgroundColor: "#22c55e" }} />
          Live
        </span>
      );
    case "reconnecting":
      return (
        <span style={containerStyle} title="Reconnecting to WebSocket...">
          <span
            style={{
              ...indicatorStyle,
              backgroundColor: "#f59e0b",
              animation: "pulse 1s infinite",
            }}
          />
          Reconnecting
        </span>
      );
    case "polling":
      return (
        <span
          style={containerStyle}
          title="Using polling fallback (4s interval)"
        >
          <span style={{ ...indicatorStyle, backgroundColor: "#6b7280" }} />
          Polling
        </span>
      );
    case "disconnected":
      return (
        <span
          style={containerStyle}
          title="Disconnected - using polling fallback"
        >
          <span style={{ ...indicatorStyle, backgroundColor: "#ef4444" }} />
          Disconnected
        </span>
      );
    default:
      return null;
  }
}

function formatDuration(ms?: number): string {
  if (!ms || ms <= 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.round(ms / 100) / 10;
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  return `${minutes}m ${Math.round(seconds % 60)}s`;
}

export function JobList({
  jobs,
  failedJobs,
  error,
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
}: JobListProps) {
  const [jumpInputValue, setJumpInputValue] = useState(currentPage.toString());

  useEffect(() => {
    setJumpInputValue(currentPage.toString());
  }, [currentPage]);

  const maxPage = Math.max(1, Math.ceil(totalJobs / jobsPerPage));

  return (
    <section className="panel">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 12,
          flexWrap: "wrap",
        }}
      >
        <div>
          <h2>Recent Runs</h2>
          <div style={{ fontSize: 13, color: "#666" }}>
            Inspect recent executions, queue progression, and terminal failure
            context from one place.
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <ConnectionIndicator state={connectionState} />
          <button type="button" className="secondary" onClick={onRefresh}>
            Refresh
          </button>
        </div>
      </div>
      {error ? <p className="error">{error}</p> : null}

      <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginTop: 12 }}>
        {[
          { value: "", label: "All" },
          { value: "queued", label: "Queued" },
          { value: "running", label: "Running" },
          { value: "failed", label: "Failed" },
          { value: "succeeded", label: "Succeeded" },
          { value: "canceled", label: "Canceled" },
        ].map((filter) => (
          <button
            key={filter.label}
            type="button"
            className={statusFilter === filter.value ? "" : "secondary"}
            onClick={() =>
              onStatusFilterChange(filter.value as JobListProps["statusFilter"])
            }
          >
            {filter.label}
          </button>
        ))}
      </div>

      {failedJobs.length > 0 ? (
        <div className="panel" style={{ marginTop: 12, background: "#fff7f7" }}>
          <strong>Recent Failures</strong>
          <ul style={{ marginTop: 8, paddingLeft: 20 }}>
            {failedJobs.slice(0, 5).map((job) => (
              <li key={job.id}>
                <code>{job.id}</code>{" "}
                {job.run?.failure ? (
                  <>
                    <strong>{job.run.failure.category}</strong>:{" "}
                    {job.run.failure.summary}
                  </>
                ) : (
                  job.error
                )}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {totalJobs > jobsPerPage ? (
        <div className="pagination-controls" style={{ marginTop: 12 }}>
          <button
            type="button"
            disabled={currentPage <= 1}
            onClick={() => onPageChange(currentPage - 1)}
          >
            Previous
          </button>

          <span className="pagination-info">
            Page {currentPage} of {maxPage} ({totalJobs} total runs)
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
              onChange={(e) => {
                const page = parseInt(e.target.value, 10);
                if (Number.isInteger(page) && page >= 1 && page <= maxPage) {
                  setJumpInputValue(e.target.value);
                }
              }}
            />
            <button
              type="button"
              onClick={() => {
                const page = parseInt(jumpInputValue, 10);
                if (page >= 1 && page <= maxPage) {
                  onPageChange(page);
                }
              }}
            >
              Go
            </button>
          </div>
        </div>
      ) : null}

      <div className="job-list" style={{ marginTop: 12 }}>
        {jobs.length === 0 ? (
          <div>No runs found for the current filter.</div>
        ) : (
          jobs.map((job) => (
            <div key={job.id} className="job-item">
              <div>{job.id}</div>
              <div>
                <span className={`badge ${getJobStatusBadgeClass(job.status)}`}>
                  {job.status}
                </span>{" "}
                {job.kind}
                {job.dependencyStatus && job.dependencyStatus !== "ready" ? (
                  <span
                    className={`badge ${getJobStatusBadgeClass(job.dependencyStatus)}`}
                    style={{ marginLeft: 8 }}
                    title={
                      job.dependencyStatus === "pending"
                        ? "Waiting for dependencies"
                        : "Dependency failed"
                    }
                  >
                    deps: {job.dependencyStatus}
                  </span>
                ) : null}
                {job.chainId ? (
                  <span
                    className="badge"
                    style={{
                      marginLeft: 8,
                      backgroundColor: "#e0e7ff",
                      color: "#4338ca",
                    }}
                    title={`Part of chain ${job.chainId}`}
                  >
                    chain
                  </span>
                ) : null}
              </div>
              {job.dependsOn && job.dependsOn.length > 0 ? (
                <div style={{ fontSize: 12, color: "#666" }}>
                  Depends on: {job.dependsOn.slice(0, 3).join(", ")}
                  {job.dependsOn.length > 3
                    ? ` +${job.dependsOn.length - 3} more`
                    : ""}
                </div>
              ) : null}
              <div style={{ fontSize: 12, color: "#666" }}>
                Wait: {formatDuration(job.run?.waitMs)} · Run:{" "}
                {formatDuration(job.run?.runMs)} · Total:{" "}
                {formatDuration(job.run?.totalMs)}
              </div>
              {job.run?.queue ? (
                <div style={{ fontSize: 12, color: "#666" }}>
                  Batch {job.run.queue.index}/{job.run.queue.total} ·{" "}
                  {job.run.queue.completed} complete · {job.run.queue.percent}%
                </div>
              ) : null}
              <div style={{ fontSize: 12, color: "#666" }}>
                Updated: {job.updatedAt}
              </div>
              {job.run?.failure ? (
                <div style={{ fontSize: 12, color: "#b91c1c" }}>
                  <strong>{job.run.failure.category}</strong>:{" "}
                  {job.run.failure.summary}
                </div>
              ) : job.error ? (
                <div>Error: {job.error}</div>
              ) : null}
              <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
                {job.status === "succeeded" ? (
                  <button
                    type="button"
                    className="secondary"
                    onClick={() => onViewResults(job.id ?? "", "jsonl", 1)}
                  >
                    View Results
                  </button>
                ) : null}
                {job.status === "queued" || job.status === "running" ? (
                  <button
                    type="button"
                    className="secondary"
                    onClick={() => onCancel(job.id ?? "")}
                  >
                    Cancel
                  </button>
                ) : null}
                <button
                  type="button"
                  className="secondary"
                  onClick={() => onDelete(job.id ?? "")}
                  style={{ color: "#ff6b6b" }}
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
