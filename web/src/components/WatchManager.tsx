/**
 * Purpose: Coordinate watch-management actions and persisted inspection workflows for the Automation route.
 * Responsibilities: Manage create/edit form state, manual checks, delete confirmation, and the watch history modal workflow.
 * Scope: Watch route coordination only; API calls arrive through props and detailed presentation stays in subcomponents.
 * Usage: Render from the watches automation container with authoritative watch data and callbacks.
 * Invariants/Assumptions: Empty watch state should still suggest a next step, only one edit form is open at a time, and persisted history remains the source of truth for post-check inspection.
 */

import { useCallback, useState } from "react";
import type { Watch, WatchCheckInspection, WatchCheckResult } from "../api";
import type { WatchManagerProps } from "../types/watch";
import { useWatchForm } from "../hooks/useWatchForm";
import { ActionEmptyState } from "./ActionEmptyState";
import { WatchList } from "./watches/WatchList";
import { WatchForm } from "./watches/WatchForm";
import { CheckResultModal } from "./watches/CheckResultModal";
import { WatchHistoryModal } from "./watches/WatchHistoryModal";

const WATCH_HISTORY_PAGE_SIZE = 10;

export function WatchManager({
  watches,
  onRefresh,
  onCreate,
  onUpdate,
  onDelete,
  onCheck,
  onLoadHistory,
  onLoadHistoryDetail,
  loading,
}: WatchManagerProps) {
  const [showForm, setShowForm] = useState(false);
  const [checkResult, setCheckResult] = useState<WatchCheckResult | null>(null);
  const [checkInspection, setCheckInspection] =
    useState<WatchCheckInspection | null>(null);
  const [checkingId, setCheckingId] = useState<string | null>(null);
  const [historyLoadingId, setHistoryLoadingId] = useState<string | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [historyWatch, setHistoryWatch] = useState<Watch | null>(null);
  const [historyRecords, setHistoryRecords] = useState<WatchCheckInspection[]>(
    [],
  );
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyOffset, setHistoryOffset] = useState(0);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [selectedHistoryCheck, setSelectedHistoryCheck] =
    useState<WatchCheckInspection | null>(null);
  const [selectedHistoryCheckLoading, setSelectedHistoryCheckLoading] =
    useState(false);

  const {
    formData,
    formError,
    formSubmitting,
    editingId,
    setFormDataPartial,
    resetForm,
    initFormForEdit,
    submitForm,
  } = useWatchForm();

  const loadHistoryDetail = useCallback(
    async (
      watchId: string,
      checkId: string,
      fallback: WatchCheckInspection | null,
    ) => {
      setSelectedHistoryCheckLoading(true);
      if (fallback) {
        setSelectedHistoryCheck(fallback);
      }
      try {
        const detail = await onLoadHistoryDetail(watchId, checkId);
        if (detail) {
          setSelectedHistoryCheck(detail);
        }
      } finally {
        setSelectedHistoryCheckLoading(false);
      }
    },
    [onLoadHistoryDetail],
  );

  const loadHistoryPage = useCallback(
    async (watchItem: Watch, offset: number, preferredCheckId?: string) => {
      setHistoryWatch(watchItem);
      setHistoryLoading(true);
      setHistoryLoadingId(watchItem.id);
      try {
        const response = await onLoadHistory(
          watchItem.id,
          WATCH_HISTORY_PAGE_SIZE,
          offset,
        );
        const records = response?.checks || [];
        setHistoryRecords(records);
        setHistoryTotal(response?.total ?? 0);
        setHistoryOffset(response?.offset ?? offset);
        const preferredRecord =
          records.find((record) => record.id === preferredCheckId) ||
          records[0] ||
          null;
        setSelectedHistoryCheck(preferredRecord);
        if (preferredRecord) {
          await loadHistoryDetail(
            watchItem.id,
            preferredRecord.id,
            preferredRecord,
          );
        }
      } finally {
        setHistoryLoading(false);
        setHistoryLoadingId(null);
      }
    },
    [loadHistoryDetail, onLoadHistory],
  );

  const handleCreateClick = useCallback(() => {
    resetForm();
    setShowForm(true);
  }, [resetForm]);

  const handleEditClick = useCallback(
    (watch: Watch) => {
      initFormForEdit(watch);
      setShowForm(true);
    },
    [initFormForEdit],
  );

  const handleCloseForm = useCallback(() => {
    setShowForm(false);
    resetForm();
  }, [resetForm]);

  const handleSubmit = useCallback(async () => {
    const success = await submitForm(onCreate, onUpdate);
    if (success) {
      setShowForm(false);
      onRefresh();
    }
  }, [submitForm, onCreate, onUpdate, onRefresh]);

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await onDelete(id);
        setDeleteConfirmId(null);
        if (historyWatch?.id === id) {
          setHistoryWatch(null);
          setHistoryRecords([]);
          setSelectedHistoryCheck(null);
        }
        onRefresh();
      } catch (err) {
        console.error("Failed to delete watch:", err);
      }
    },
    [historyWatch?.id, onDelete, onRefresh],
  );

  const handleCheck = useCallback(
    async (watch: Watch) => {
      setCheckingId(watch.id);
      setCheckResult(null);
      setCheckInspection(null);
      try {
        const result = await onCheck(watch.id);
        if (result) {
          setCheckResult(result);
          if (result.checkId) {
            const detail = await onLoadHistoryDetail(watch.id, result.checkId);
            if (detail) {
              setCheckInspection(detail);
            }
          }
        }
      } catch (err) {
        console.error("Check failed:", err);
      } finally {
        setCheckingId(null);
      }
    },
    [onCheck, onLoadHistoryDetail],
  );

  const handleOpenHistory = useCallback(
    async (watch: Watch, preferredCheckId?: string) => {
      await loadHistoryPage(watch, 0, preferredCheckId);
    },
    [loadHistoryPage],
  );

  const handleOpenHistoryFromCheck = useCallback(
    async (checkId: string) => {
      const watchItem = watches.find(
        (item) => item.id === checkResult?.watchId,
      );
      if (!watchItem) {
        return;
      }
      setCheckResult(null);
      setCheckInspection(null);
      await handleOpenHistory(watchItem, checkId);
    },
    [checkResult?.watchId, handleOpenHistory, watches],
  );

  const handleHistoryPageChange = useCallback(
    async (offset: number) => {
      if (!historyWatch) {
        return;
      }
      await loadHistoryPage(historyWatch, Math.max(0, offset));
    },
    [historyWatch, loadHistoryPage],
  );

  const handleHistorySelect = useCallback(
    async (checkId: string) => {
      if (!historyWatch) {
        return;
      }
      const fallback =
        historyRecords.find((record) => record.id === checkId) || null;
      await loadHistoryDetail(historyWatch.id, checkId, fallback);
    },
    [historyRecords, historyWatch, loadHistoryDetail],
  );

  return (
    <div className="panel">
      <div
        className="row"
        style={{
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <h2 style={{ margin: 0 }}>Watch Monitoring</h2>
        <div className="row" style={{ gap: 8 }}>
          <button
            type="button"
            onClick={onRefresh}
            disabled={loading}
            className="secondary"
          >
            {loading ? "Loading..." : "Refresh"}
          </button>
          <button type="button" onClick={handleCreateClick}>
            + Add Watch
          </button>
        </div>
      </div>

      {watches.length === 0 && !loading ? (
        <ActionEmptyState
          eyebrow="Automation"
          title="No watches configured yet"
          description="Add a watch to monitor a page for content changes and inspect every saved check from the same workspace."
          actions={[
            { label: "Add watch", onClick: handleCreateClick },
            { label: "Refresh", onClick: onRefresh, tone: "secondary" },
          ]}
        />
      ) : (
        <WatchList
          watches={watches}
          checkingId={checkingId}
          historyLoadingId={historyLoadingId}
          deleteConfirmId={deleteConfirmId}
          onEdit={handleEditClick}
          onDelete={handleDelete}
          onCheck={handleCheck}
          onHistory={(watch) => {
            void handleOpenHistory(watch);
          }}
          onDeleteConfirm={setDeleteConfirmId}
        />
      )}

      {checkResult ? (
        <CheckResultModal
          result={checkResult}
          inspection={checkInspection}
          onClose={() => {
            setCheckResult(null);
            setCheckInspection(null);
          }}
          onOpenHistory={(checkId) => {
            void handleOpenHistoryFromCheck(checkId);
          }}
        />
      ) : null}

      {historyWatch ? (
        <WatchHistoryModal
          watch={historyWatch}
          records={historyRecords}
          total={historyTotal}
          limit={WATCH_HISTORY_PAGE_SIZE}
          offset={historyOffset}
          loading={historyLoading}
          selectedCheck={selectedHistoryCheck}
          selectedCheckLoading={selectedHistoryCheckLoading}
          onClose={() => {
            setHistoryWatch(null);
            setHistoryRecords([]);
            setSelectedHistoryCheck(null);
            setHistoryOffset(0);
            setHistoryTotal(0);
          }}
          onSelectCheck={(checkId) => {
            void handleHistorySelect(checkId);
          }}
          onPageChange={(offset) => {
            void handleHistoryPageChange(offset);
          }}
        />
      ) : null}

      {showForm ? (
        <WatchForm
          formData={formData}
          formError={formError}
          formSubmitting={formSubmitting}
          isEditing={!!editingId}
          onChange={setFormDataPartial}
          onSubmit={handleSubmit}
          onCancel={handleCloseForm}
        />
      ) : null}
    </div>
  );
}
