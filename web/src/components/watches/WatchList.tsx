/**
 * WatchList Component
 *
 * Renders the table of watches with headers and individual watch rows.
 * Handles empty state and delegates row rendering to WatchListItem.
 *
 * This component does NOT handle:
 * - API calls for watch operations
 * - Modal dialogs (those are handled by parent)
 * - Sorting logic (watches should be pre-sorted by parent)
 *
 * @module components/watches/WatchList
 */

import { useMemo } from "react";
import type { WatchListProps } from "../../types/watch";
import { WatchListItem } from "./WatchListItem";

/**
 * Table component for displaying the list of watches
 */
export function WatchList({
  watches,
  checkingId,
  historyLoadingId,
  deleteConfirmId,
  onEdit,
  onDelete,
  onCheck,
  onHistory,
  onDeleteConfirm,
}: WatchListProps) {
  const sortedWatches = useMemo(() => {
    return [...watches].sort((a, b) => {
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
    });
  }, [watches]);

  return (
    <div className="watch-list">
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr style={{ borderBottom: "1px solid var(--stroke)" }}>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>URL</th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>Status</th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>Interval</th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>Changes</th>
            <th style={{ textAlign: "left", padding: "8px 12px" }}>
              Last Checked
            </th>
            <th style={{ textAlign: "right", padding: "8px 12px" }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {sortedWatches.map((watch) => (
            <WatchListItem
              key={watch.id}
              watch={watch}
              isChecking={checkingId === watch.id}
              isHistoryLoading={historyLoadingId === watch.id}
              isDeleting={deleteConfirmId === watch.id}
              onEdit={() => onEdit(watch)}
              onDelete={() => onDelete(watch.id)}
              onCheck={() => onCheck(watch)}
              onHistory={() => onHistory(watch)}
              onDeleteConfirm={() =>
                onDeleteConfirm(deleteConfirmId === watch.id ? null : watch.id)
              }
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}
