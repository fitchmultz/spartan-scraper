/**
 * ExportScheduleList Component
 *
 * Renders the table of export schedules with headers and individual rows.
 * Handles empty state and delegates row rendering to ExportScheduleListItem.
 *
 * This component does NOT handle:
 * - API calls for export schedule operations
 * - Modal dialogs (those are handled by parent)
 * - Sorting logic (schedules should be pre-sorted by parent)
 *
 * @module components/export-schedules/ExportScheduleList
 */

import { useMemo } from "react";
import type { ExportScheduleListProps } from "../../types/export-schedule";
import { ExportScheduleListItem } from "./ExportScheduleListItem";

/**
 * Table component for displaying the list of export schedules
 */
export function ExportScheduleList({
  schedules,
  historyLoadingId,
  deleteConfirmId,
  onEdit,
  onDelete,
  onToggleEnabled,
  onViewHistory,
  onDeleteConfirm,
}: ExportScheduleListProps) {
  const sortedSchedules = useMemo(() => {
    return [...schedules].sort((a, b) => {
      const dateA = a.created_at ? new Date(a.created_at).getTime() : 0;
      const dateB = b.created_at ? new Date(b.created_at).getTime() : 0;
      return dateB - dateA;
    });
  }, [schedules]);

  return (
    <div className="export-schedule-list">
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr style={{ borderBottom: "1px solid var(--stroke)" }}>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>Name</th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>Status</th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>Filters</th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>
              Destination
            </th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>Format</th>
            <th style={{ textAlign: "right", padding: "8px 12px" }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {sortedSchedules.map((schedule) => (
            <ExportScheduleListItem
              key={schedule.id}
              schedule={schedule}
              isHistoryLoading={historyLoadingId === schedule.id}
              isDeleting={deleteConfirmId === schedule.id}
              onEdit={() => onEdit(schedule)}
              onDelete={() => onDelete(schedule.id)}
              onToggleEnabled={(enabled) =>
                onToggleEnabled(schedule.id, enabled)
              }
              onViewHistory={() => onViewHistory(schedule)}
              onDeleteConfirm={() =>
                onDeleteConfirm(
                  deleteConfirmId === schedule.id ? null : schedule.id,
                )
              }
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}
