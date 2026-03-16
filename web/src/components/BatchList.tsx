/**
 * Purpose: Render paginated batch summaries and on-demand batch detail rows for the Web UI.
 * Responsibilities: Display aggregate batch status, highlight recent submissions, expose refresh/cancel/detail actions, and provide pagination controls.
 * Scope: Presentation only; data fetching and state management stay in the batches container/hook.
 * Usage: Render from the batch route container with authoritative batch summaries and optional cached job details.
 * Invariants/Assumptions: Batch rows show summary data immediately; job tables appear only after details are loaded for a batch.
 */
import { useState, useCallback, useEffect } from "react";
import type { BatchJobStats, Job } from "../api";
import {
  calculateBatchProgress,
  getStatusClass,
  isTerminalStatus,
} from "../lib/batch-utils";
import { formatDateTime } from "../lib/formatting";

interface BatchListProps {
  batches: BatchEntry[];
  jobs?: Map<string, Job[]>;
  total: number;
  limit: number;
  offset: number;
  highlightedBatchId?: string | null;
  onViewStatus: (batchId: string) => void | Promise<void>;
  onCancel: (batchId: string) => void;
  onRefresh: () => void;
  onPageChange: (offset: number) => void;
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

export function BatchList({
  batches,
  jobs,
  total,
  limit,
  offset,
  highlightedBatchId,
  onViewStatus,
  onCancel,
  onRefresh,
  onPageChange,
  loading,
}: BatchListProps) {
  const [expandedBatch, setExpandedBatch] = useState<string | null>(null);

  useEffect(() => {
    if (highlightedBatchId) {
      setExpandedBatch(highlightedBatchId);
    }
  }, [highlightedBatchId]);

  const toggleExpand = useCallback((batchId: string) => {
    setExpandedBatch((current) => (current === batchId ? null : batchId));
  }, []);

  const hasMore = offset + limit < total;
  const hasPrev = offset > 0;
  const currentPage = limit > 0 ? Math.floor(offset / limit) + 1 : 1;
  const totalPages = limit > 0 ? Math.max(1, Math.ceil(total / limit)) : 1;

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
        <h2>Batch Jobs ({total})</h2>
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
          const progress = calculateBatchProgress(batch.stats, batch.jobCount);
          const isExpanded = expandedBatch === batch.id;
          const batchJobs = jobs?.get(batch.id) || [];
          const isHighlighted = batch.id === highlightedBatchId;

          return (
            <div
              key={batch.id}
              id={`batch-${batch.id}`}
              className="batch-item"
              style={{
                border: isHighlighted
                  ? "1px solid var(--accent)"
                  : "1px solid var(--border)",
                borderRadius: 8,
                padding: 16,
                background: "var(--panel-bg)",
                boxShadow: isHighlighted
                  ? "0 0 0 1px color-mix(in srgb, var(--accent) 28%, transparent), 0 20px 40px rgba(0, 0, 0, 0.14)"
                  : undefined,
              }}
            >
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
                    className={`badge ${getStatusClass(batch.status)}`}
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
                  {isHighlighted ? (
                    <span className="badge running">Just submitted</span>
                  ) : null}
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
                      <div>Created: {formatDateTime(batch.createdAt)}</div>
                      <div>Updated: {formatDateTime(batch.updatedAt)}</div>
                    </div>
                    <div style={{ display: "flex", gap: 8 }}>
                      {!isTerminalStatus(batch.status) && (
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
                          void onViewStatus(batch.id);
                        }}
                      >
                        {batchJobs.length > 0
                          ? "Refresh Details"
                          : "View Details"}
                      </button>
                    </div>
                  </div>

                  {batchJobs.length > 0 ? (
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
                                    className={`badge ${getStatusClass(job.status)}`}
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
                  ) : (
                    <p
                      style={{
                        margin: 0,
                        color: "var(--text-muted)",
                        fontSize: 13,
                      }}
                    >
                      Load batch details to inspect individual jobs.
                    </p>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {total > 0 && (
        <div
          className="row"
          style={{
            justifyContent: "space-between",
            alignItems: "center",
            marginTop: 16,
            paddingTop: 16,
            borderTop: "1px solid var(--stroke)",
          }}
        >
          <div style={{ fontSize: 14, color: "var(--muted)" }}>
            Showing {offset + 1}-{Math.min(offset + batches.length, total)} of{" "}
            {total}
          </div>
          <div className="row" style={{ gap: 8 }}>
            <button
              type="button"
              onClick={() => onPageChange(Math.max(0, offset - limit))}
              disabled={!hasPrev || loading}
              className="secondary"
            >
              Previous
            </button>
            <span
              style={{
                fontSize: 14,
                padding: "8px 12px",
                color: "var(--muted)",
              }}
            >
              Page {currentPage} of {totalPages}
            </span>
            <button
              type="button"
              onClick={() => onPageChange(offset + limit)}
              disabled={!hasMore || loading}
              className="secondary"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
