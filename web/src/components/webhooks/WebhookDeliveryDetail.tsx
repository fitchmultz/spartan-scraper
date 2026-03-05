/**
 * WebhookDeliveryDetail Component
 *
 * Modal component for displaying detailed information about a webhook delivery.
 * Shows full metadata, request payload, response body, retry history, and error details.
 *
 * This component does NOT handle:
 * - API calls
 * - State management
 *
 * @module components/webhooks/WebhookDeliveryDetail
 */

import type { WebhookDeliveryDetailProps } from "../../types/webhook";

/**
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return "-";
  try {
    const date = new Date(dateStr);
    return date.toLocaleString();
  } catch {
    return dateStr;
  }
}

/**
 * Format JSON for display
 */
function formatJson(data: unknown): string {
  try {
    return JSON.stringify(data, null, 2);
  } catch {
    return String(data);
  }
}

/**
 * Get status color based on delivery status
 */
function getStatusColor(status?: string): string {
  switch (status?.toLowerCase()) {
    case "delivered":
      return "var(--success, #22c55e)";
    case "failed":
      return "var(--error, #ef4444)";
    case "pending":
      return "var(--warning, #f59e0b)";
    default:
      return "var(--muted, #6b7280)";
  }
}

export function WebhookDeliveryDetail({
  delivery,
  loading,
  onClose,
}: WebhookDeliveryDetailProps) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Webhook delivery details"
      className="modal-overlay"
      style={{
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: "rgba(0, 0, 0, 0.5)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        padding: 20,
      }}
      onClick={onClose}
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          onClose();
        }
      }}
      tabIndex={-1}
    >
      <div
        role="document"
        className="modal-content"
        style={{
          backgroundColor: "var(--bg)",
          borderRadius: 8,
          border: "1px solid var(--stroke)",
          maxWidth: 800,
          width: "100%",
          maxHeight: "90vh",
          overflow: "auto",
          boxShadow: "0 20px 25px -5px rgba(0, 0, 0, 0.1)",
        }}
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          // Stop propagation for keyboard events inside the modal content
          // to prevent closing the modal when interacting with content
          e.stopPropagation();
        }}
      >
        {/* Header */}
        <div
          style={{
            padding: "20px 24px",
            borderBottom: "1px solid var(--stroke)",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <div>
            <h2 style={{ margin: 0, fontSize: 20 }}>Webhook Delivery Detail</h2>
            <p
              style={{
                margin: "4px 0 0 0",
                fontSize: 14,
                color: "var(--muted)",
              }}
            >
              ID: {delivery.id}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="secondary"
            style={{ padding: "8px 16px" }}
          >
            Close
          </button>
        </div>

        {/* Content */}
        <div style={{ padding: 24 }}>
          {loading ? (
            <div style={{ textAlign: "center", padding: 40 }}>
              Loading delivery details...
            </div>
          ) : (
            <>
              {/* Status Section */}
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
                  gap: 16,
                  marginBottom: 24,
                }}
              >
                <div
                  style={{
                    padding: 16,
                    backgroundColor: "var(--surface)",
                    borderRadius: 8,
                    border: "1px solid var(--stroke)",
                  }}
                >
                  <div
                    style={{
                      fontSize: 12,
                      color: "var(--muted)",
                      textTransform: "uppercase",
                      marginBottom: 4,
                    }}
                  >
                    Status
                  </div>
                  <div
                    style={{
                      fontSize: 18,
                      fontWeight: 600,
                      color: getStatusColor(delivery.status),
                      textTransform: "uppercase",
                    }}
                  >
                    {delivery.status || "unknown"}
                  </div>
                </div>

                <div
                  style={{
                    padding: 16,
                    backgroundColor: "var(--surface)",
                    borderRadius: 8,
                    border: "1px solid var(--stroke)",
                  }}
                >
                  <div
                    style={{
                      fontSize: 12,
                      color: "var(--muted)",
                      textTransform: "uppercase",
                      marginBottom: 4,
                    }}
                  >
                    Attempts
                  </div>
                  <div style={{ fontSize: 18, fontWeight: 600 }}>
                    {delivery.attempts ?? 0}
                  </div>
                </div>

                {delivery.responseCode !== undefined && (
                  <div
                    style={{
                      padding: 16,
                      backgroundColor: "var(--surface)",
                      borderRadius: 8,
                      border: "1px solid var(--stroke)",
                    }}
                  >
                    <div
                      style={{
                        fontSize: 12,
                        color: "var(--muted)",
                        textTransform: "uppercase",
                        marginBottom: 4,
                      }}
                    >
                      Response Code
                    </div>
                    <div style={{ fontSize: 18, fontWeight: 600 }}>
                      {delivery.responseCode}
                    </div>
                  </div>
                )}
              </div>

              {/* Metadata Section */}
              <div
                style={{
                  marginBottom: 24,
                }}
              >
                <h3
                  style={{
                    margin: "0 0 12px 0",
                    fontSize: 16,
                    fontWeight: 600,
                  }}
                >
                  Delivery Metadata
                </h3>
                <div
                  style={{
                    display: "grid",
                    gap: 8,
                    fontSize: 14,
                  }}
                >
                  <div className="row" style={{ gap: 8 }}>
                    <span style={{ color: "var(--muted)", minWidth: 100 }}>
                      Event ID:
                    </span>
                    <code>{delivery.eventId || "-"}</code>
                  </div>
                  <div className="row" style={{ gap: 8 }}>
                    <span style={{ color: "var(--muted)", minWidth: 100 }}>
                      Event Type:
                    </span>
                    <span>{delivery.eventType || "-"}</span>
                  </div>
                  <div className="row" style={{ gap: 8 }}>
                    <span style={{ color: "var(--muted)", minWidth: 100 }}>
                      Job ID:
                    </span>
                    <code>{delivery.jobId || "-"}</code>
                  </div>
                  <div className="row" style={{ gap: 8 }}>
                    <span style={{ color: "var(--muted)", minWidth: 100 }}>
                      URL:
                    </span>
                    <code
                      style={{
                        wordBreak: "break-all",
                      }}
                    >
                      {delivery.url || "-"}
                    </code>
                  </div>
                  <div className="row" style={{ gap: 8 }}>
                    <span style={{ color: "var(--muted)", minWidth: 100 }}>
                      Created:
                    </span>
                    <span>{formatDate(delivery.createdAt)}</span>
                  </div>
                  <div className="row" style={{ gap: 8 }}>
                    <span style={{ color: "var(--muted)", minWidth: 100 }}>
                      Updated:
                    </span>
                    <span>{formatDate(delivery.updatedAt)}</span>
                  </div>
                  {delivery.deliveredAt && (
                    <div className="row" style={{ gap: 8 }}>
                      <span style={{ color: "var(--muted)", minWidth: 100 }}>
                        Delivered:
                      </span>
                      <span>{formatDate(delivery.deliveredAt)}</span>
                    </div>
                  )}
                </div>
              </div>

              {/* Error Section */}
              {delivery.lastError && (
                <div
                  style={{
                    marginBottom: 24,
                  }}
                >
                  <h3
                    style={{
                      margin: "0 0 12px 0",
                      fontSize: 16,
                      fontWeight: 600,
                      color: "var(--error)",
                    }}
                  >
                    Error Details
                  </h3>
                  <pre
                    style={{
                      margin: 0,
                      padding: 16,
                      backgroundColor: "rgba(239, 68, 68, 0.1)",
                      border: "1px solid var(--error)",
                      borderRadius: 8,
                      fontSize: 13,
                      color: "var(--error)",
                      overflow: "auto",
                      maxHeight: 200,
                    }}
                  >
                    {delivery.lastError}
                  </pre>
                </div>
              )}

              {/* Technical Details */}
              <div>
                <h3
                  style={{
                    margin: "0 0 12px 0",
                    fontSize: 16,
                    fontWeight: 600,
                  }}
                >
                  Technical Details
                </h3>
                <pre
                  style={{
                    margin: 0,
                    padding: 16,
                    backgroundColor: "var(--surface)",
                    border: "1px solid var(--stroke)",
                    borderRadius: 8,
                    fontSize: 13,
                    overflow: "auto",
                    maxHeight: 300,
                  }}
                >
                  {formatJson(delivery)}
                </pre>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
