/**
 * Purpose: Render the export schedule list item UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import type { ExportScheduleListItemProps } from "../../types/export-schedule";
import { getEnabledStatusTone } from "../../lib/status-display";
import {
  formatDestination,
  formatExportShapeSummary,
  formatExportTransformSummary,
  formatFilters,
} from "../../lib/export-schedule-utils";
import { StatusPill } from "../StatusPill";

/**
 * Single export schedule row component with status badge and action buttons
 */
export function ExportScheduleListItem({
  schedule,
  isHistoryLoading,
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
        <StatusPill
          label={schedule.enabled ? "Enabled" : "Disabled"}
          tone={getEnabledStatusTone(schedule.enabled)}
        />
      </td>
      <td style={{ padding: "12px", fontSize: 13 }}>
        {formatFilters(schedule.filters)}
      </td>
      <td style={{ padding: "12px", fontSize: 13 }}>
        {formatDestination(schedule)}
      </td>
      <td style={{ padding: "12px", fontSize: 13 }}>
        <div>{schedule.export?.format?.toUpperCase() || "-"}</div>
        <div style={{ fontSize: 12, color: "var(--muted)", marginTop: 2 }}>
          {schedule.export?.transform?.expression
            ? formatExportTransformSummary(schedule.export.transform)
            : formatExportShapeSummary(schedule.export?.shape)}
        </div>
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
            disabled={isHistoryLoading}
            className="secondary"
            style={{ padding: "6px 12px", fontSize: 12 }}
            title="View export history"
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
