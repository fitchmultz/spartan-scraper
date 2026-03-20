/**
 * ExportScheduleHistory Component
 *
 * Purpose:
 * - Render guided export outcome inspection for a schedule inside a modal.
 *
 * Responsibilities:
 * - Show export status, narrative, artifact metadata, failures, and next steps.
 * - Preserve offset-based pagination driven by parent callbacks.
 * - Keep history readable even when outcomes include no artifact or failure details.
 *
 * Scope:
 * - Modal presentation only; data loading and pagination state stay in parent components.
 *
 * Usage:
 * - Render from ExportScheduleManager once schedule history has been loaded.
 *
 * Invariants/Assumptions:
 * - Records are already sanitized and transport-safe.
 * - Unknown statuses and failure categories should still render gracefully.
 */

import type { ExportScheduleHistoryProps } from "../../types/export-schedule";
import { formatDateTime } from "../../lib/formatting";
import { formatFileSize } from "../../lib/export-schedule-utils";
import { getExportHistoryStatusTone } from "../../lib/status-display";
import { ActionEmptyState } from "../ActionEmptyState";
import { CapabilityActionList } from "../CapabilityActionList";
import { StatusPill } from "../StatusPill";

function formatFailureCategory(category?: string) {
  if (!category) {
    return "Unknown";
  }
  return category
    .split("-")
    .join(" ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export function ExportScheduleHistory({
  scheduleName,
  records,
  total,
  limit,
  offset,
  loading,
  onClose,
  onPageChange,
}: ExportScheduleHistoryProps) {
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.max(1, Math.ceil(total / limit));

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        backgroundColor: "rgba(0, 0, 0, 0.7)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        padding: 20,
      }}
    >
      <div
        className="panel"
        style={{
          maxWidth: 960,
          width: "100%",
          maxHeight: "90vh",
          overflow: "auto",
        }}
      >
        <div
          className="row"
          style={{
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 16,
          }}
        >
          <div>
            <h3 style={{ margin: 0 }}>Export History: {scheduleName}</h3>
            <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>
              Inspect outcomes, understand failures quickly, and follow the next
              recovery step without leaving the history view.
            </p>
          </div>
          <button type="button" onClick={onClose} className="secondary">
            Close
          </button>
        </div>

        {loading ? (
          <div role="status" aria-live="polite" style={{ padding: 24 }}>
            <ActionEmptyState
              eyebrow="History"
              title="Loading export history"
              description="Fetching recent export outcomes for this schedule."
            />
          </div>
        ) : records.length === 0 ? (
          <div style={{ padding: 24 }}>
            <ActionEmptyState
              eyebrow="History"
              title="No export history found"
              description="History will appear when jobs matching this schedule are exported."
            />
          </div>
        ) : (
          <>
            <div
              className="row"
              style={{
                justifyContent: "space-between",
                alignItems: "center",
                marginBottom: 16,
                fontSize: 13,
                color: "var(--muted)",
              }}
            >
              <span>
                Showing {offset + 1}-{Math.min(offset + records.length, total)}{" "}
                of {total}
              </span>
              <span>
                Page {currentPage} of {totalPages}
              </span>
            </div>

            <div style={{ display: "grid", gap: 12 }}>
              {records.map((record) => (
                <article
                  key={record.id}
                  className="panel"
                  style={{
                    padding: 16,
                    border: "1px solid var(--stroke)",
                    background: "rgba(255, 255, 255, 0.02)",
                  }}
                >
                  <div
                    className="row"
                    style={{
                      justifyContent: "space-between",
                      alignItems: "flex-start",
                      gap: 16,
                    }}
                  >
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div
                        className="row"
                        style={{
                          alignItems: "center",
                          gap: 8,
                          marginBottom: 8,
                        }}
                      >
                        <StatusPill
                          label={record.status}
                          tone={getExportHistoryStatusTone(record.status)}
                        />
                        <span style={{ color: "var(--muted)", fontSize: 12 }}>
                          {record.trigger} export
                        </span>
                      </div>
                      <h4 style={{ margin: 0 }}>{record.title}</h4>
                      <p style={{ margin: "8px 0 0", color: "var(--muted)" }}>
                        {record.message}
                      </p>
                    </div>
                    <div
                      style={{
                        fontFamily: "monospace",
                        fontSize: 12,
                        color: "var(--muted)",
                        textAlign: "right",
                      }}
                    >
                      <div>{record.id}</div>
                      <div>{record.jobId}</div>
                    </div>
                  </div>

                  <div
                    style={{
                      display: "grid",
                      gridTemplateColumns:
                        "repeat(auto-fit, minmax(180px, 1fr))",
                      gap: 12,
                      marginTop: 16,
                    }}
                  >
                    <div>
                      <strong>Requested format</strong>
                      <div>{record.request.format}</div>
                    </div>
                    <div>
                      <strong>Destination</strong>
                      <div style={{ wordBreak: "break-word" }}>
                        {record.destination || "-"}
                      </div>
                    </div>
                    <div>
                      <strong>Exported at</strong>
                      <div>{formatDateTime(record.exportedAt)}</div>
                    </div>
                    <div>
                      <strong>Completed at</strong>
                      <div>
                        {record.completedAt
                          ? formatDateTime(record.completedAt)
                          : "-"}
                      </div>
                    </div>
                    <div>
                      <strong>Retries</strong>
                      <div>{record.retryCount}</div>
                    </div>
                    <div>
                      <strong>Artifact</strong>
                      <div>
                        {record.artifact
                          ? `${record.artifact.filename} · ${formatFileSize(record.artifact.size)}`
                          : "Not available"}
                      </div>
                    </div>
                  </div>

                  {record.failure ? (
                    <div
                      style={{
                        marginTop: 16,
                        padding: 12,
                        borderRadius: 12,
                        border: "1px solid rgba(239, 68, 68, 0.3)",
                        background: "rgba(239, 68, 68, 0.08)",
                      }}
                    >
                      <strong>
                        {formatFailureCategory(record.failure.category)} issue
                      </strong>
                      <p style={{ margin: "8px 0 0" }}>
                        {record.failure.summary}
                      </p>
                      <div style={{ color: "var(--muted)", fontSize: 12 }}>
                        {record.failure.retryable
                          ? "This outcome looks retryable."
                          : "This outcome needs a config or data change before retrying."}
                      </div>
                    </div>
                  ) : null}

                  {record.actions?.length ? (
                    <div style={{ marginTop: 16 }}>
                      <strong>Recommended next steps</strong>
                      <div style={{ marginTop: 8 }}>
                        <CapabilityActionList
                          actions={record.actions}
                          onNavigate={(path) => {
                            window.location.assign(path);
                          }}
                          onRefresh={async () => undefined}
                        />
                      </div>
                    </div>
                  ) : null}
                </article>
              ))}
            </div>

            {totalPages > 1 ? (
              <div
                className="row"
                style={{
                  justifyContent: "center",
                  gap: 8,
                  marginTop: 16,
                }}
              >
                <button
                  type="button"
                  onClick={() => onPageChange(offset - limit)}
                  disabled={offset === 0}
                  className="secondary"
                >
                  Previous
                </button>
                {Array.from({ length: totalPages }, (_, i) => i + 1).map(
                  (page) => (
                    <button
                      key={page}
                      type="button"
                      onClick={() => onPageChange((page - 1) * limit)}
                      className={page === currentPage ? "" : "secondary"}
                      style={{
                        minWidth: 36,
                        padding: "6px 12px",
                      }}
                    >
                      {page}
                    </button>
                  ),
                )}
                <button
                  type="button"
                  onClick={() => onPageChange(offset + limit)}
                  disabled={offset + limit >= total}
                  className="secondary"
                >
                  Next
                </button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}
