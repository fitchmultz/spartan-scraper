/**
 * Purpose: Render the watch list UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
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
