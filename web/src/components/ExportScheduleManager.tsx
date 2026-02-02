/**
 * Export Schedule Manager Component
 *
 * Provides UI for managing automated export schedules. Supports creating,
 * editing, deleting, and viewing export history. Displays schedule status,
 * filters, and destinations.
 *
 * This is a coordination layer component that delegates to:
 * - ExportScheduleList for displaying the schedules table
 * - ExportScheduleForm for create/edit modal
 * - ExportScheduleHistory for viewing history
 * - useExportScheduleForm hook for form state management
 *
 * @module ExportScheduleManager
 */

import { useState, useCallback } from "react";
import type { ExportSchedule, ExportHistoryRecord } from "../api";
import type { ExportScheduleManagerProps } from "../types/export-schedule";
import { useExportScheduleForm } from "../hooks/useExportScheduleForm";
import { ExportScheduleList } from "./export-schedules/ExportScheduleList";
import { ExportScheduleForm } from "./export-schedules/ExportScheduleForm";
import { ExportScheduleHistory } from "./export-schedules/ExportScheduleHistory";

const HISTORY_PAGE_SIZE = 10;

export function ExportScheduleManager({
  schedules,
  onRefresh,
  onCreate,
  onUpdate,
  onDelete,
  onToggleEnabled,
  onGetHistory,
  loading,
}: ExportScheduleManagerProps) {
  const [showForm, setShowForm] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [historySchedule, setHistorySchedule] = useState<ExportSchedule | null>(
    null,
  );
  const [historyRecords, setHistoryRecords] = useState<ExportHistoryRecord[]>(
    [],
  );
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyOffset, setHistoryOffset] = useState(0);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);

  const {
    formData,
    formError,
    formSubmitting,
    editingId,
    setFormDataPartial,
    resetForm,
    initFormForEdit,
    submitForm,
  } = useExportScheduleForm();

  const handleCreateClick = useCallback(() => {
    resetForm();
    setShowForm(true);
  }, [resetForm]);

  const handleEditClick = useCallback(
    (schedule: ExportSchedule) => {
      initFormForEdit(schedule);
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
        onRefresh();
      } catch (err) {
        console.error("Failed to delete export schedule:", err);
      }
    },
    [onDelete, onRefresh],
  );

  const handleToggleEnabled = useCallback(
    async (id: string, enabled: boolean) => {
      try {
        await onToggleEnabled(id, enabled);
        onRefresh();
      } catch (err) {
        console.error("Failed to toggle export schedule:", err);
      }
    },
    [onToggleEnabled, onRefresh],
  );

  const loadHistory = useCallback(
    async (schedule: ExportSchedule, offset = 0) => {
      setHistoryLoading(true);
      setHistorySchedule(schedule);
      setHistoryOffset(offset);
      try {
        const result = await onGetHistory(
          schedule.id,
          HISTORY_PAGE_SIZE,
          offset,
        );
        setHistoryRecords(result.records);
        setHistoryTotal(result.total);
        setShowHistory(true);
      } catch (err) {
        console.error("Failed to load export history:", err);
      } finally {
        setHistoryLoading(false);
      }
    },
    [onGetHistory],
  );

  const handleViewHistory = useCallback(
    (schedule: ExportSchedule) => {
      loadHistory(schedule, 0);
    },
    [loadHistory],
  );

  const handleHistoryPageChange = useCallback(
    (offset: number) => {
      if (historySchedule) {
        loadHistory(historySchedule, offset);
      }
    },
    [loadHistory, historySchedule],
  );

  const handleCloseHistory = useCallback(() => {
    setShowHistory(false);
    setHistorySchedule(null);
    setHistoryRecords([]);
    setHistoryTotal(0);
    setHistoryOffset(0);
  }, []);

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
        <h2 style={{ margin: 0 }}>Export Schedules</h2>
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
            + Add Schedule
          </button>
        </div>
      </div>

      <p style={{ color: "var(--muted)", marginBottom: 16, fontSize: 14 }}>
        Automatically export job results when jobs complete matching specified
        filter criteria.
      </p>

      {schedules.length === 0 && !loading ? (
        <div
          style={{
            textAlign: "center",
            padding: "40px 20px",
            color: "var(--muted)",
          }}
        >
          <p>No export schedules configured yet.</p>
          <p>
            Click &quot;Add Schedule&quot; to create automated exports for your
            jobs.
          </p>
        </div>
      ) : (
        <ExportScheduleList
          schedules={schedules}
          deleteConfirmId={deleteConfirmId}
          onEdit={handleEditClick}
          onDelete={handleDelete}
          onToggleEnabled={handleToggleEnabled}
          onViewHistory={handleViewHistory}
          onDeleteConfirm={setDeleteConfirmId}
        />
      )}

      {showForm && (
        <ExportScheduleForm
          formData={formData}
          formError={formError}
          formSubmitting={formSubmitting}
          isEditing={!!editingId}
          onChange={setFormDataPartial}
          onSubmit={handleSubmit}
          onCancel={handleCloseForm}
        />
      )}

      {showHistory && historySchedule && (
        <ExportScheduleHistory
          scheduleName={historySchedule.name}
          records={historyRecords}
          total={historyTotal}
          limit={HISTORY_PAGE_SIZE}
          offset={historyOffset}
          loading={historyLoading}
          onClose={handleCloseHistory}
          onPageChange={handleHistoryPageChange}
        />
      )}
    </div>
  );
}
