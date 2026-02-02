/**
 * WebhookDeliveryList Component
 *
 * Table component for displaying webhook deliveries with status badges,
 * timestamps, and action buttons.
 *
 * This component does NOT handle:
 * - API calls
 * - Modal dialogs
 * - Pagination logic
 *
 * @module components/webhooks/WebhookDeliveryList
 */

import type { WebhookDeliveryListProps } from "../../types/webhook";

/**
 * Get status badge color based on delivery status
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

/**
 * Get status background color for badges
 */
function getStatusBgColor(status?: string): string {
  switch (status?.toLowerCase()) {
    case "delivered":
      return "rgba(34, 197, 94, 0.1)";
    case "failed":
      return "rgba(239, 68, 68, 0.1)";
    case "pending":
      return "rgba(245, 158, 11, 0.1)";
    default:
      return "rgba(107, 114, 128, 0.1)";
  }
}

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
 * Truncate ID for display
 */
function truncateId(id?: string, length = 8): string {
  if (!id) return "-";
  if (id.length <= length * 2 + 3) return id;
  return `${id.slice(0, length)}...${id.slice(-length)}`;
}

/**
 * Truncate URL for display
 */
function truncateUrl(url?: string, maxLength = 50): string {
  if (!url) return "-";
  if (url.length <= maxLength) return url;
  return `${url.slice(0, maxLength)}...`;
}

/**
 * Copy text to clipboard
 */
async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch (err) {
    console.error("Failed to copy to clipboard:", err);
  }
}

export function WebhookDeliveryList({
  deliveries,
  onViewDetail,
}: WebhookDeliveryListProps) {
  return (
    <div
      className="webhook-delivery-list"
      style={{
        overflowX: "auto",
      }}
    >
      <table
        style={{
          width: "100%",
          borderCollapse: "collapse",
          fontSize: 14,
        }}
      >
        <thead>
          <tr
            style={{
              borderBottom: "1px solid var(--stroke)",
              backgroundColor: "var(--surface)",
            }}
          >
            <th
              style={{
                textAlign: "left",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              ID
            </th>
            <th
              style={{
                textAlign: "left",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              Event Type
            </th>
            <th
              style={{
                textAlign: "left",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              Job ID
            </th>
            <th
              style={{
                textAlign: "left",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              URL
            </th>
            <th
              style={{
                textAlign: "left",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              Status
            </th>
            <th
              style={{
                textAlign: "center",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              Attempts
            </th>
            <th
              style={{
                textAlign: "left",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              Created
            </th>
            <th
              style={{
                textAlign: "right",
                padding: "12px 16px",
                fontWeight: 600,
              }}
            >
              Actions
            </th>
          </tr>
        </thead>
        <tbody>
          {deliveries.map((delivery) => (
            <tr
              key={delivery.id}
              style={{
                borderBottom: "1px solid var(--stroke)",
              }}
            >
              <td style={{ padding: "12px 16px" }}>
                <div className="row" style={{ gap: 8, alignItems: "center" }}>
                  <code style={{ fontSize: 12 }}>
                    {truncateId(delivery.id)}
                  </code>
                  <button
                    type="button"
                    onClick={() => delivery.id && copyToClipboard(delivery.id)}
                    className="secondary"
                    style={{
                      padding: "2px 6px",
                      fontSize: 11,
                    }}
                    title="Copy full ID"
                  >
                    Copy
                  </button>
                </div>
              </td>
              <td style={{ padding: "12px 16px" }}>
                {delivery.eventType || "-"}
              </td>
              <td style={{ padding: "12px 16px" }}>
                {delivery.jobId ? (
                  <button
                    type="button"
                    onClick={() => {
                      // Scroll to jobs section
                      const jobsSection = document.getElementById("jobs");
                      if (jobsSection) {
                        jobsSection.scrollIntoView({ behavior: "smooth" });
                      }
                    }}
                    style={{
                      color: "var(--accent)",
                      textDecoration: "none",
                      background: "none",
                      border: "none",
                      padding: 0,
                      cursor: "pointer",
                      fontFamily: "inherit",
                      fontSize: "inherit",
                    }}
                    title={delivery.jobId}
                  >
                    {truncateId(delivery.jobId)}
                  </button>
                ) : (
                  "-"
                )}
              </td>
              <td
                style={{
                  padding: "12px 16px",
                  maxWidth: 200,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
                title={delivery.url}
              >
                {truncateUrl(delivery.url)}
              </td>
              <td style={{ padding: "12px 16px" }}>
                <span
                  style={{
                    display: "inline-block",
                    padding: "4px 8px",
                    borderRadius: 4,
                    fontSize: 12,
                    fontWeight: 600,
                    textTransform: "uppercase",
                    color: getStatusColor(delivery.status),
                    backgroundColor: getStatusBgColor(delivery.status),
                  }}
                >
                  {delivery.status || "unknown"}
                </span>
              </td>
              <td
                style={{
                  padding: "12px 16px",
                  textAlign: "center",
                }}
              >
                {delivery.attempts ?? 0}
              </td>
              <td style={{ padding: "12px 16px" }}>
                {formatDate(delivery.createdAt)}
              </td>
              <td style={{ padding: "12px 16px", textAlign: "right" }}>
                <button
                  type="button"
                  onClick={() => onViewDetail(delivery)}
                  className="secondary"
                  style={{ padding: "6px 12px", fontSize: 12 }}
                >
                  View Details
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
