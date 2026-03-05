/**
 * WatchListItem Component
 *
 * Renders a single watch row in the watches table.
 * Displays watch status, URL, selector, interval, changes, and action buttons.
 *
 * This component does NOT handle:
 * - API calls for watch operations
 * - State management for the watch list
 * - Modal dialogs (those are handled by parent)
 *
 * @module components/watches/WatchListItem
 */

import type { WatchListItemProps } from "../../types/watch";
import { formatDuration, formatDate } from "../../lib/watch-utils";

/**
 * Single watch row component with status badge and action buttons
 */
export function WatchListItem({
  watch,
  isChecking,
  isDeleting,
  onEdit,
  onDelete,
  onCheck,
  onDeleteConfirm,
}: WatchListItemProps) {
  return (
    <tr style={{ borderBottom: "1px solid var(--stroke)" }}>
      <td style={{ padding: "12px" }}>
        <div style={{ fontWeight: 500 }}>{watch.url}</div>
        {watch.selector && (
          <div
            style={{
              fontSize: 12,
              color: "var(--muted)",
              marginTop: 2,
            }}
          >
            Selector: {watch.selector}
          </div>
        )}
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
              watch.status === "active"
                ? "rgba(34, 197, 94, 0.15)"
                : "rgba(156, 163, 175, 0.15)",
            color: watch.status === "active" ? "#22c55e" : "var(--muted)",
          }}
        >
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: "50%",
              backgroundColor:
                watch.status === "active" ? "#22c55e" : "var(--muted)",
            }}
          />
          {watch.status}
        </span>
      </td>
      <td style={{ padding: "12px" }}>
        {formatDuration(watch.intervalSeconds)}
      </td>
      <td style={{ padding: "12px" }}>
        <span
          style={{
            fontWeight: 600,
            color: (watch.changeCount || 0) > 0 ? "var(--accent)" : "inherit",
          }}
        >
          {watch.changeCount || 0}
        </span>
      </td>
      <td style={{ padding: "12px", fontSize: 13 }}>
        {formatDate(watch.lastCheckedAt)}
      </td>
      <td style={{ padding: "12px", textAlign: "right" }}>
        <div className="row" style={{ gap: 6, justifyContent: "flex-end" }}>
          <button
            type="button"
            onClick={onCheck}
            disabled={isChecking}
            className="secondary"
            style={{ padding: "6px 12px", fontSize: 12 }}
            title="Check now"
          >
            {isChecking ? "Checking..." : "Check"}
          </button>
          <button
            type="button"
            onClick={onEdit}
            className="secondary"
            style={{ padding: "6px 12px", fontSize: 12 }}
          >
            Edit
          </button>
          {isDeleting ? (
            <>
              <button
                type="button"
                onClick={onDelete}
                style={{
                  padding: "6px 12px",
                  fontSize: 12,
                  backgroundColor: "#ef4444",
                }}
              >
                Confirm
              </button>
              <button
                type="button"
                onClick={onDeleteConfirm}
                className="secondary"
                style={{ padding: "6px 12px", fontSize: 12 }}
              >
                Cancel
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={onDeleteConfirm}
              className="secondary"
              style={{ padding: "6px 12px", fontSize: 12 }}
            >
              Delete
            </button>
          )}
        </div>
      </td>
    </tr>
  );
}
