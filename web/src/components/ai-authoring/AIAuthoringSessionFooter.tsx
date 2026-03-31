/**
 * Purpose: Render the shared footer action row for AI authoring generator and debugger modals.
 * Responsibilities: Keep close/discard/reset/run/save controls consistent while letting callers supply the exact labels and disabled states.
 * Scope: Shared footer presentation only; callers own the underlying async actions and busy-state derivation.
 * Usage: Mount as the footer for an AI authoring modal and pass route-specific labels plus action handlers.
 * Invariants/Assumptions: Close is always available, discard only appears when a draft exists, and run/save actions switch based on whether the session already has attempts.
 */

interface AIAuthoringSessionFooterProps {
  onClose: () => void;
  hasSessionDraft: boolean;
  hasAttempts: boolean;
  onDiscardSession: () => void;
  onResetSession: () => void;
  onRetry: () => void;
  onSave: () => void;
  onRun: () => void;
  discardDisabled: boolean;
  resetDisabled: boolean;
  retryDisabled: boolean;
  saveDisabled: boolean;
  runDisabled: boolean;
  actionTitle?: string;
  runLabel: string;
  runningLabel: string;
  retryLabel: string;
  retryingLabel: string;
  saveLabel: string;
  savingLabel: string;
  isRunning: boolean;
  isSaving: boolean;
}

export function AIAuthoringSessionFooter({
  onClose,
  hasSessionDraft,
  hasAttempts,
  onDiscardSession,
  onResetSession,
  onRetry,
  onSave,
  onRun,
  discardDisabled,
  resetDisabled,
  retryDisabled,
  saveDisabled,
  runDisabled,
  actionTitle,
  runLabel,
  runningLabel,
  retryLabel,
  retryingLabel,
  saveLabel,
  savingLabel,
  isRunning,
  isSaving,
}: AIAuthoringSessionFooterProps) {
  return (
    <div className="modal-footer gap-3">
      <button type="button" className="button-secondary" onClick={onClose}>
        Close
      </button>
      {hasSessionDraft ? (
        <button
          type="button"
          className="button-secondary"
          onClick={onDiscardSession}
          disabled={discardDisabled}
        >
          Discard session
        </button>
      ) : null}
      {hasAttempts ? (
        <>
          <button
            type="button"
            className="button-secondary"
            onClick={onResetSession}
            disabled={resetDisabled}
          >
            Reset session
          </button>
          <button
            type="button"
            className="button-secondary"
            onClick={onRetry}
            disabled={retryDisabled}
            title={actionTitle}
          >
            {isRunning ? retryingLabel : retryLabel}
          </button>
          <button
            type="button"
            className="button-primary"
            onClick={onSave}
            disabled={saveDisabled}
            title={actionTitle}
          >
            {isSaving ? savingLabel : saveLabel}
          </button>
        </>
      ) : (
        <button
          type="button"
          className="button-primary"
          onClick={onRun}
          disabled={runDisabled}
          title={actionTitle}
        >
          {isRunning ? runningLabel : runLabel}
        </button>
      )}
    </div>
  );
}
