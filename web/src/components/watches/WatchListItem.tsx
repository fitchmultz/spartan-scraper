/**
 * Purpose: Render the watch list item UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import type { WatchListItemProps } from "../../types/watch";
import { formatDateTime, formatSecondsAsDuration } from "../../lib/formatting";
import { getWatchStatusTone } from "../../lib/status-display";
import { StatusPill } from "../StatusPill";

/**
 * Single watch row component with status badge and action buttons
 */
export function WatchListItem({
  watch,
  isChecking,
  isHistoryLoading,
  isDeleting,
  onEdit,
  onDelete,
  onCheck,
  onHistory,
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
        {watch.jobTrigger && (
          <div
            style={{
              fontSize: 12,
              color: "var(--muted)",
              marginTop: 2,
            }}
          >
            Trigger: {watch.jobTrigger.kind}
          </div>
        )}
      </td>
      <td style={{ padding: "12px" }}>
        <StatusPill
          label={watch.status}
          tone={getWatchStatusTone(watch.status)}
        />
      </td>
      <td style={{ padding: "12px" }}>
        {formatSecondsAsDuration(watch.intervalSeconds)}
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
        {formatDateTime(watch.lastCheckedAt, "Never")}
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
            onClick={onHistory}
            disabled={isHistoryLoading}
            className="secondary"
            style={{ padding: "6px 12px", fontSize: 12 }}
          >
            {isHistoryLoading ? "Loading..." : "History"}
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
