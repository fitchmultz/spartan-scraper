/**
 * Batch List Component
 *
 * Displays batch jobs with their status and aggregated statistics.
 * Provides actions to view details, cancel running batches.
 *
 * @module BatchList
 */
import { useState, useCallback } from "react";
import type { BatchJobStats, Job } from "../api";

interface BatchListProps {
  batches: BatchEntry[];
  jobs?: Map<string, Job[]>; // batch ID -> jobs
  onViewStatus: (batchId: string) => void;
  onCancel: (batchId: string) => void;
  onRefresh: () => void;
  loading: boolean;
}

export type BatchEntry = {
  id: string;
  kind: "scrape" | "crawl" | "research";
  status:
    | "pending"
    | "processing"
    | "completed"
    | "failed"
    | "partial"
    | "canceled";
  jobCount: number;
  stats: BatchJobStats;
  createdAt: string;
  updatedAt: string;
};

function calculateProgress(stats: BatchJobStats, total: number): number {
  if (total === 0) return 0;
  const completed = stats.succeeded + stats.failed + stats.canceled;
  return Math.round((completed / total) * 100);
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString();
}

function getStatusBadgeClass(status: BatchEntry["status"]): string {
  switch (status) {
    case "completed":
      return "success";
    case "processing":
      return "running";
    case "failed":
    case "canceled":
      return "failed";
    case "partial":
      return "warning";
    default:
      return "";
  }
}

export function BatchList({
  batches,
  jobs,
  onViewStatus,
  onCancel,
  onRefresh,
  loading,
}: BatchListProps) {
  const [expandedBatch, setExpandedBatch] = useState<string | null>(null);

  const toggleExpand = useCallback((batchId: string) => {
    setExpandedBatch((current) => (current === batchId ? null : batchId));
  }, []);

  const isTerminal = (status: BatchEntry["status"]): boolean => {
    return ["completed", "failed", "partial", "canceled"].includes(status);
  };

  if (batches.length === 0) {
    return (
      <div className="panel">
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 16,
          }}
        >
          <h2>Batch Jobs</h2>
          <button
            type="button"
            className="secondary"
            onClick={onRefresh}
            disabled={loading}
          >
            {loading ? "Loading..." : "Refresh"}
          </button>
        </div>
        <p
          style={{
            color: "var(--text-muted)",
            textAlign: "center",
            padding: 32,
          }}
        >
          No batch jobs yet. Use the Batch form to submit jobs.
        </p>
      </div>
    );
  }

  return (
    <div className="panel">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <h2>Batch Jobs ({batches.length})</h2>
        <button
          type="button"
          className="secondary"
          onClick={onRefresh}
          disabled={loading}
        >
          {loading ? "Loading..." : "Refresh"}
        </button>
      </div>

      <div
        className="batch-list"
        style={{ display: "flex", flexDirection: "column", gap: 12 }}
      >
        {batches.map((batch) => {
          const progress = calculateProgress(batch.stats, batch.jobCount);
          const isExpanded = expandedBatch === batch.id;
          const batchJobs = jobs?.get(batch.id) || [];

          return (
            <div
              key={batch.id}
              className="batch-item"
              style={{
                border: "1px solid var(--border)",
                borderRadius: 8,
                padding: 16,
                background: "var(--panel-bg)",
              }}
            >
              {/* Header */}
              <button
                type="button"
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  width: "100%",
                  background: "none",
                  border: "none",
                  padding: 0,
                  cursor: "pointer",
                  textAlign: "left",
                }}
                onClick={() => toggleExpand(batch.id)}
                aria-expanded={isExpanded}
                aria-label={`Batch ${batch.id.slice(0, 8)} ${batch.status}`}
              >
                <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                  <span
                    className={`badge ${getStatusBadgeClass(batch.status)}`}
                    style={{
                      padding: "4px 8px",
                      borderRadius: 4,
                      fontSize: 12,
                      fontWeight: 600,
                      textTransform: "uppercase",
                    }}
                  >
                    {batch.status}
                  </span>
                  <span style={{ fontWeight: 600 }}>
                    {batch.id.slice(0, 8)}...
                  </span>
                  <span
                    style={{
                      textTransform: "capitalize",
                      color: "var(--text-muted)",
                      fontSize: 14,
                    }}
                  >
                    {batch.kind}
                  </span>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                  <span style={{ color: "var(--text-muted)", fontSize: 14 }}>
                    {batch.stats.succeeded}/{batch.jobCount} done
                  </span>
                  <span style={{ fontSize: 12 }}>{isExpanded ? "▼" : "▶"}</span>
                </div>
              </button>

              {/* Progress bar */}
              <div
                style={{
                  marginTop: 12,
                  height: 6,
                  background: "var(--border)",
                  borderRadius: 3,
                  overflow: "hidden",
                }}
              >
                <div
                  style={{
                    width: `${progress}%`,
                    height: "100%",
                    background:
                      batch.status === "failed" || batch.status === "canceled"
                        ? "var(--error)"
                        : batch.status === "completed"
                          ? "var(--success)"
                          : batch.status === "partial"
                            ? "var(--warning)"
                            : "var(--accent)",
                    transition: "width 0.3s ease",
                  }}
                />
              </div>

              {/* Stats */}
              <div
                style={{
                  marginTop: 12,
                  display: "flex",
                  gap: 16,
                  fontSize: 13,
                  color: "var(--text-muted)",
                }}
              >
                <span>Queued: {batch.stats.queued}</span>
                <span>Running: {batch.stats.running}</span>
                <span style={{ color: "var(--success)" }}>
                  Succeeded: {batch.stats.succeeded}
                </span>
                <span style={{ color: "var(--error)" }}>
                  Failed: {batch.stats.failed}
                </span>
                <span>Canceled: {batch.stats.canceled}</span>
              </div>

              {/* Expanded details */}
              {isExpanded && (
                <div
                  style={{
                    marginTop: 16,
                    paddingTop: 16,
                    borderTop: "1px solid var(--border)",
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      marginBottom: 12,
                    }}
                  >
                    <div style={{ fontSize: 13, color: "var(--text-muted)" }}>
                      <div>Created: {formatDate(batch.createdAt)}</div>
                      <div>Updated: {formatDate(batch.updatedAt)}</div>
                    </div>
                    <div style={{ display: "flex", gap: 8 }}>
                      {!isTerminal(batch.status) && (
                        <button
                          type="button"
                          className="secondary"
                          onClick={(e) => {
                            e.stopPropagation();
                            onCancel(batch.id);
                          }}
                        >
                          Cancel
                        </button>
                      )}
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          onViewStatus(batch.id);
                        }}
                      >
                        View Details
                      </button>
                    </div>
                  </div>

                  {/* Job list */}
                  {batchJobs.length > 0 && (
                    <div style={{ marginTop: 12 }}>
                      <h4 style={{ fontSize: 14, marginBottom: 8 }}>Jobs</h4>
                      <div
                        style={{
                          maxHeight: 200,
                          overflow: "auto",
                          border: "1px solid var(--border)",
                          borderRadius: 4,
                        }}
                      >
                        <table style={{ width: "100%", fontSize: 13 }}>
                          <thead>
                            <tr style={{ background: "var(--bg)" }}>
                              <th style={{ textAlign: "left", padding: 8 }}>
                                ID
                              </th>
                              <th style={{ textAlign: "left", padding: 8 }}>
                                Status
                              </th>
                            </tr>
                          </thead>
                          <tbody>
                            {batchJobs.map((job) => (
                              <tr key={job.id}>
                                <td style={{ padding: 8 }}>
                                  {job.id.slice(0, 16)}...
                                </td>
                                <td style={{ padding: 8 }}>
                                  <span
                                    className={`badge ${getStatusBadgeClass(job.status as BatchEntry["status"])}`}
                                  >
                                    {job.status}
                                  </span>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
