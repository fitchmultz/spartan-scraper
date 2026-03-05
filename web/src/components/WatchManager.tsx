/**
 * Watch Manager Component
 *
 * Provides UI for managing content change monitoring watches. Supports creating,
 * editing, deleting, and manually checking watches. Displays watch status,
 * change counts, and next run times.
 *
 * This is a coordination layer component that delegates to:
 * - WatchList for displaying the watches table
 * - WatchForm for create/edit modal
 * - CheckResultModal for displaying check results
 * - useWatchForm hook for form state management
 *
 * @module WatchManager
 */

import { useState, useCallback } from "react";
import type { Watch, WatchCheckResult } from "../api";
import type { WatchManagerProps } from "../types/watch";
import { useWatchForm } from "../hooks/useWatchForm";
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
        <div
          style={{
            textAlign: "center",
            padding: "40px 20px",
            color: "var(--muted)",
          }}
        >
          <p>No watches configured yet.</p>
          <p>Click "Add Watch" to start monitoring content changes.</p>
        </div>
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
