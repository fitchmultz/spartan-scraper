/**
 * ExportScheduleHistory Component
 *
 * Renders a modal showing export history for a schedule.
 * Displays a table with job ID, status, destination, timestamps, and stats.
 *
 * This component does NOT handle:
 * - API calls for fetching history (parent handles via onGetHistory)
 * - State management for pagination (controlled via props)
 *
 * @module components/export-schedules/ExportScheduleHistory
 */

import type { ExportScheduleHistoryProps } from "../../types/export-schedule";
import { formatDateTime } from "../../lib/formatting";
import { formatFileSize } from "../../lib/export-schedule-utils";

/**
 * Modal component for displaying export history
 */
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
  const totalPages = Math.ceil(total / limit);

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
          maxWidth: 900,
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
          <h3 style={{ margin: 0 }}>Export History: {scheduleName}</h3>
          <button type="button" onClick={onClose} className="secondary">
            Close
          </button>
        </div>

        {loading ? (
          <div style={{ textAlign: "center", padding: 40 }}>Loading...</div>
        ) : records.length === 0 ? (
          <div
            style={{
              textAlign: "center",
              padding: "40px 20px",
              color: "var(--muted)",
            }}
          >
            <p>No export history found.</p>
            <p>
              History will appear when jobs matching this schedule are exported.
            </p>
          </div>
        ) : (
          <>
            <div
              className="row"
              style={{
                justifyContent: "space-between",
                alignItems: "center",
                marginBottom: 12,
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

            <table style={{ width: "100%", borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ borderBottom: "1px solid var(--stroke)" }}>
                  <th style={{ textAlign: "left", padding: "8px 12px" }}>
                    Job ID
                  </th>
                  <th style={{ textAlign: "left", padding: "8px 12px" }}>
                    Status
                  </th>
                  <th style={{ textAlign: "left", padding: "8px 12px" }}>
                    Destination
                  </th>
                  <th style={{ textAlign: "left", padding: "8px 12px" }}>
                    Exported At
                  </th>
                  <th style={{ textAlign: "left", padding: "8px 12px" }}>
                    Records
                  </th>
                  <th style={{ textAlign: "left", padding: "8px 12px" }}>
                    Size
                  </th>
                </tr>
              </thead>
              <tbody>
                {records.map((record) => (
                  <tr
                    key={record.id}
                    style={{ borderBottom: "1px solid var(--stroke)" }}
                  >
                    <td
                      style={{
                        padding: "12px",
                        fontFamily: "monospace",
                        fontSize: 12,
                      }}
                    >
                      {record.job_id?.substring(0, 12)}...
                    </td>
                    <td style={{ padding: "12px" }}>
                      <span
                        style={{
                          display: "inline-flex",
                          alignItems: "center",
                          gap: 6,
                          padding: "4px 10px",
                          borderRadius: 12,
                          fontSize: 12,
                          fontWeight: 500,
                          backgroundColor:
                            record.status === "success"
                              ? "rgba(34, 197, 94, 0.15)"
                              : record.status === "pending"
                                ? "rgba(234, 179, 8, 0.15)"
                                : "rgba(239, 68, 68, 0.15)",
                          color:
                            record.status === "success"
                              ? "#22c55e"
                              : record.status === "pending"
                                ? "#eab308"
                                : "#ef4444",
                        }}
                      >
                        <span
                          style={{
                            width: 6,
                            height: 6,
                            borderRadius: "50%",
                            backgroundColor:
                              record.status === "success"
                                ? "#22c55e"
                                : record.status === "pending"
                                  ? "#eab308"
                                  : "#ef4444",
                          }}
                        />
                        {record.status}
                      </span>
                    </td>
                    <td
                      style={{
                        padding: "12px",
                        fontSize: 13,
                        maxWidth: 200,
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {record.destination || "-"}
                    </td>
                    <td style={{ padding: "12px", fontSize: 13 }}>
                      {formatDateTime(record.exported_at)}
                    </td>
                    <td style={{ padding: "12px", fontSize: 13 }}>
                      {record.record_count ?? "-"}
                    </td>
                    <td style={{ padding: "12px", fontSize: 13 }}>
                      {formatFileSize(record.export_size)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {totalPages > 1 && (
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
            )}
          </>
        )}
      </div>
    </div>
  );
}
