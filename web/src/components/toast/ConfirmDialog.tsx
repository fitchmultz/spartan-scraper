/**
 * Purpose: Render the shared confirmation dialog used to replace browser-native `confirm()` calls.
 * Responsibilities: Present destructive-action context, expose accessible confirm/cancel controls, and close on backdrop click or Escape.
 * Scope: Notification-owned confirmation UI only.
 * Usage: Rendered internally by `ToastProvider`; callers invoke it indirectly via `useToast().confirm()`.
 * Invariants/Assumptions: At most one confirmation request is active, the confirm button receives initial focus, and dismissing the dialog always resolves the pending request.
 */

import { useEffect, useEffectEvent, useRef, type MouseEvent } from "react";
import type { ToastTone } from "./ToastProvider";

export interface ConfirmDialogOptions {
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: Extract<ToastTone, "success" | "info" | "warning" | "error">;
}

interface ConfirmDialogRequest extends ConfirmDialogOptions {
  id: string;
}

interface ConfirmDialogProps {
  request: ConfirmDialogRequest | null;
  onCancel: () => void;
  onConfirm: () => void;
}

export function ConfirmDialog({
  request,
  onCancel,
  onConfirm,
}: ConfirmDialogProps) {
  const confirmButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!request) {
      return;
    }

    confirmButtonRef.current?.focus();
  }, [request]);

  const handleEscape = useEffectEvent((event: KeyboardEvent) => {
    if (!request || event.key !== "Escape") {
      return;
    }

    event.preventDefault();
    onCancel();
  });

  useEffect(() => {
    if (!request) {
      return;
    }

    window.addEventListener("keydown", handleEscape);
    return () => window.removeEventListener("keydown", handleEscape);
  }, [request]);

  if (!request) {
    return null;
  }

  const handleBackdropClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      onCancel();
    }
  };

  const tone = request.tone ?? "warning";

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: backdrop click closes the shared confirmation dialog.
    <div
      className="toast-confirm__overlay"
      onClick={handleBackdropClick}
      role="presentation"
    >
      <div
        className={`toast-confirm toast-confirm--${tone}`}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="toast-confirm-title"
        aria-describedby={
          request.description ? "toast-confirm-description" : undefined
        }
      >
        <div className="toast-confirm__eyebrow">Confirmation required</div>
        <h2 id="toast-confirm-title" className="toast-confirm__title">
          {request.title}
        </h2>
        {request.description ? (
          <p
            id="toast-confirm-description"
            className="toast-confirm__description"
          >
            {request.description}
          </p>
        ) : null}
        <div className="toast-confirm__actions">
          <button type="button" className="secondary" onClick={onCancel}>
            {request.cancelLabel ?? "Cancel"}
          </button>
          <button
            ref={confirmButtonRef}
            type="button"
            className={`toast-confirm__confirm toast-confirm__confirm--${tone}`}
            onClick={onConfirm}
          >
            {request.confirmLabel ?? "Confirm"}
          </button>
        </div>
      </div>
    </div>
  );
}
