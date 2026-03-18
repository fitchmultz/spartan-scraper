/**
 * Purpose: Coordinate watch-management actions for the Automation route.
 * Responsibilities: Load create/edit form state, coordinate manual checks and deletions, and render a guided empty state when no watches exist yet.
 * Scope: Watch route coordination only; API calls arrive through props and detailed presentation stays in subcomponents.
 * Usage: Render from the watches automation container with authoritative watch data and callbacks.
 * Invariants/Assumptions: Empty watch state should still suggest a next step, only one edit form is open at a time, and check results stay visible until dismissed.
 */

import { useState, useCallback } from "react";
import type { Watch, WatchCheckResult } from "../api";
import type { WatchManagerProps } from "../types/watch";
import { useWatchForm } from "../hooks/useWatchForm";
import { ActionEmptyState } from "./ActionEmptyState";
import { WatchList } from "./watches/WatchList";
import { WatchForm } from "./watches/WatchForm";
import { CheckResultModal } from "./watches/CheckResultModal";

export function WatchManager({
  watches,
  onRefresh,
  onCreate,
  onUpdate,
  onDelete,
  onCheck,
  loading,
}: WatchManagerProps) {
  const [showForm, setShowForm] = useState(false);
  const [checkResult, setCheckResult] = useState<WatchCheckResult | null>(null);
  const [checkingId, setCheckingId] = useState<string | null>(null);
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
  } = useWatchForm();

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
        onRefresh();
      } catch (err) {
        console.error("Failed to delete watch:", err);
      }
    },
    [onDelete, onRefresh],
  );

  const handleCheck = useCallback(
    async (watch: Watch) => {
      setCheckingId(watch.id);
      setCheckResult(null);
      try {
        const result = await onCheck(watch.id);
        if (result) {
          setCheckResult(result);
        }
      } catch (err) {
        console.error("Check failed:", err);
      } finally {
        setCheckingId(null);
      }
    },
    [onCheck],
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
          description="Add a watch to monitor a page for content changes and check it manually or on its configured cadence."
          actions={[
            { label: "Add watch", onClick: handleCreateClick },
            { label: "Refresh", onClick: onRefresh, tone: "secondary" },
          ]}
        />
      ) : (
        <WatchList
          watches={watches}
          checkingId={checkingId}
          deleteConfirmId={deleteConfirmId}
          onEdit={handleEditClick}
          onDelete={handleDelete}
          onCheck={handleCheck}
          onDeleteConfirm={setDeleteConfirmId}
        />
      )}

      {checkResult && (
        <CheckResultModal
          result={checkResult}
          onClose={() => setCheckResult(null)}
        />
      )}

      {showForm && (
        <WatchForm
          formData={formData}
          formError={formError}
          formSubmitting={formSubmitting}
          isEditing={!!editingId}
          onChange={setFormDataPartial}
          onSubmit={handleSubmit}
          onCancel={handleCloseForm}
        />
      )}
    </div>
  );
}
