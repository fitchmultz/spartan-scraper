/**
 * ExportScheduleListItem Component
 *
 * Renders a single export schedule row in the schedules table.
 * Displays schedule status, filters, destination, format, and action buttons.
 *
 * This component does NOT handle:
 * - API calls for export schedule operations
 * - State management for the schedule list
 * - Modal dialogs (those are handled by parent)
 *
 * @module components/export-schedules/ExportScheduleListItem
 */

import type { ExportScheduleListItemProps } from "../../types/export-schedule";
import {
  formatDestination,
  formatFilters,
} from "../../lib/export-schedule-utils";

/**
 * Single export schedule row component with status badge and action buttons
 */
export function ExportScheduleListItem({
  schedule,
  isDeleting,
  onEdit,
  onDelete,
  onToggleEnabled,
  onViewHistory,
  onDeleteConfirm,
}: ExportScheduleListItemProps) {
  return (
    <tr style={{ borderBottom: "1px solid var(--stroke)" }}>
      <td style={{ padding: "12px" }}>
        <div style={{ fontWeight: 500 }}>{schedule.name}</div>
        <div
          style={{
            fontSize: 12,
            color: "var(--muted)",
            marginTop: 2,
          }}
        >
          {schedule.id.substring(0, 8)}...
        </div>
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
            backgroundColor: schedule.enabled
              ? "rgba(34, 197, 94, 0.15)"
              : "rgba(156, 163, 175, 0.15)",
            color: schedule.enabled ? "#22c55e" : "var(--muted)",
          }}
        >
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: "50%",
              backgroundColor: schedule.enabled ? "#22c55e" : "var(--muted)",
            }}
          />
          {schedule.enabled ? "Enabled" : "Disabled"}
        </span>
      </td>
      <td style={{ padding: "12px", fontSize: 13 }}>
        {formatFilters(schedule.filters)}
      </td>
      <td style={{ padding: "12px", fontSize: 13 }}>
        {formatDestination(schedule)}
      </td>
      <td style={{ padding: "12px", fontSize: 13 }}>
        {schedule.export?.format?.toUpperCase() || "-"}
      </td>
      <td style={{ padding: "12px", textAlign: "right" }}>
        <div className="row" style={{ gap: 6, justifyContent: "flex-end" }}>
          <button
            type="button"
            onClick={() => onToggleEnabled(!schedule.enabled)}
            className="secondary"
            style={{ padding: "6px 12px", fontSize: 12 }}
            title={schedule.enabled ? "Disable schedule" : "Enable schedule"}
          >
            {schedule.enabled ? "Disable" : "Enable"}
          </button>
          <button
            type="button"
            onClick={onViewHistory}
            className="secondary"
            style={{ padding: "6px 12px", fontSize: 12 }}
            title="View export history"
          >
            History
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
