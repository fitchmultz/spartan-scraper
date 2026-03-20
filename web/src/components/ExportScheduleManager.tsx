/**
 * Purpose: Coordinate export-schedule management for the Automation route.
 * Responsibilities: Handle create/edit/delete/toggle flows, load export history, and render a guided empty state when no schedules exist yet.
 * Scope: Export-schedule route coordination only; API calls arrive through props and list/history presentation stays in subcomponents.
 * Usage: Render from the export-schedules automation container with authoritative schedules and callbacks.
 * Invariants/Assumptions: Empty schedule state should still suggest a next step, history pagination is offset-based, and only one editor modal is open at a time.
 */

import { useState, useCallback, useEffect, useRef } from "react";
import type { ExportInspection, ExportSchedule } from "../api";
import type { ExportScheduleManagerProps } from "../types/export-schedule";
import { useExportScheduleForm } from "../hooks/useExportScheduleForm";
import { ActionEmptyState } from "./ActionEmptyState";
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
  aiStatus = null,
  promotionSeed = null,
  onClearPromotionSeed,
  onOpenSourceJob,
}: ExportScheduleManagerProps) {
  const [showForm, setShowForm] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [historySchedule, setHistorySchedule] = useState<ExportSchedule | null>(
    null,
  );
  const [historyRecords, setHistoryRecords] = useState<ExportInspection[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyOffset, setHistoryOffset] = useState(0);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyLoadingId, setHistoryLoadingId] = useState<string | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const historyRequestSeqRef = useRef(0);

  const {
    formData,
    formError,
    formSubmitting,
    editingId,
    setFormDataPartial,
    resetForm,
    initFormForEdit,
    initFormFromDraft,
    submitForm,
  } = useExportScheduleForm();

  useEffect(() => {
    if (!promotionSeed) {
      return;
    }

    initFormFromDraft(promotionSeed.formData);
    setShowForm(true);
    setDeleteConfirmId(null);
    setShowHistory(false);
  }, [initFormFromDraft, promotionSeed]);

  const handleCreateClick = useCallback(() => {
    onClearPromotionSeed?.();
    resetForm();
    setShowForm(true);
  }, [onClearPromotionSeed, resetForm]);

  const handleEditClick = useCallback(
    (schedule: ExportSchedule) => {
      onClearPromotionSeed?.();
      initFormForEdit(schedule);
      setShowForm(true);
    },
    [initFormForEdit, onClearPromotionSeed],
  );

  const handleCloseForm = useCallback(() => {
    onClearPromotionSeed?.();
    setShowForm(false);
    resetForm();
  }, [onClearPromotionSeed, resetForm]);

  const handleSubmit = useCallback(async () => {
    const success = await submitForm(onCreate, onUpdate);
    if (success) {
      onClearPromotionSeed?.();
      setShowForm(false);
      onRefresh();
    }
  }, [onClearPromotionSeed, onCreate, onRefresh, onUpdate, submitForm]);

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
      const requestSeq = historyRequestSeqRef.current + 1;
      historyRequestSeqRef.current = requestSeq;
      setHistoryLoading(true);
      setHistoryLoadingId(schedule.id);
      setHistorySchedule(schedule);
      try {
        const result = await onGetHistory(
          schedule.id,
          HISTORY_PAGE_SIZE,
          offset,
        );
        if (requestSeq !== historyRequestSeqRef.current) {
          return;
        }
        setHistoryRecords(result.exports);
        setHistoryTotal(result.total);
        setHistoryOffset(result.offset ?? offset);
        setShowHistory(true);
      } catch (err) {
        if (requestSeq === historyRequestSeqRef.current) {
          console.error("Failed to load export history:", err);
        }
      } finally {
        if (requestSeq === historyRequestSeqRef.current) {
          setHistoryLoading(false);
          setHistoryLoadingId(null);
        }
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
    historyRequestSeqRef.current += 1;
    setShowHistory(false);
    setHistorySchedule(null);
    setHistoryRecords([]);
    setHistoryTotal(0);
    setHistoryOffset(0);
    setHistoryLoading(false);
    setHistoryLoadingId(null);
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
        Automatically export job results when future matching jobs complete.
      </p>

      {schedules.length === 0 && !loading ? (
        <ActionEmptyState
          eyebrow="Automation"
          title="No export schedules yet"
          description="Create a recurring export when you want completed jobs to automatically fan out into files or downstream systems."
          actions={[
            { label: "Add schedule", onClick: handleCreateClick },
            { label: "Refresh", onClick: onRefresh, tone: "secondary" },
          ]}
        />
      ) : (
        <ExportScheduleList
          schedules={schedules}
          historyLoadingId={historyLoadingId}
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
          aiStatus={aiStatus}
          promotionSeed={editingId ? null : promotionSeed}
          onClearPromotionSeed={onClearPromotionSeed}
          onOpenSourceJob={onOpenSourceJob}
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
