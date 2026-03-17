/**
 * Purpose: Render a single themed toast notification with accessibility semantics and optional actions.
 * Responsibilities: Display toast tone/title/description, auto-dismiss timed notifications, and expose action/dismiss controls.
 * Scope: Presentational toast card only.
 * Usage: Rendered by `ToastContainer` for each active toast record.
 * Invariants/Assumptions: Timed toasts reset their dismissal timer when updated, and loading toasts stay visible until explicitly changed or dismissed.
 */

import { useEffect, useMemo, useState } from "react";
import type { ToastRecord } from "./ToastProvider";

interface ToastProps {
  toast: ToastRecord;
  onDismiss: (id: string) => void;
}

function getToneLabel(tone: ToastRecord["tone"]): string {
  switch (tone) {
    case "success":
      return "Success";
    case "info":
      return "Info";
    case "warning":
      return "Warning";
    case "error":
      return "Error";
    case "loading":
      return "Loading";
  }
}

function getToneIcon(tone: ToastRecord["tone"]): string {
  switch (tone) {
    case "success":
      return "✓";
    case "info":
      return "i";
    case "warning":
      return "!";
    case "error":
      return "×";
    case "loading":
      return "…";
  }
}

export function Toast({ toast, onDismiss }: ToastProps) {
  const [isActionPending, setIsActionPending] = useState(false);

  useEffect(() => {
    if (toast.durationMs === undefined) {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      onDismiss(toast.id);
    }, toast.durationMs);

    return () => window.clearTimeout(timeoutId);
  }, [onDismiss, toast.durationMs, toast.id]);

  const role =
    toast.tone === "error" || toast.tone === "warning" ? "alert" : "status";
  const ariaLive = toast.tone === "error" ? "assertive" : "polite";
  const toneLabel = useMemo(() => getToneLabel(toast.tone), [toast.tone]);

  const handleAction = async () => {
    if (!toast.action || isActionPending) {
      return;
    }

    setIsActionPending(true);
    try {
      await toast.action.onSelect();
      onDismiss(toast.id);
    } finally {
      setIsActionPending(false);
    }
  };

  return (
    <article
      className={`toast toast--${toast.tone}`}
      role={role}
      aria-live={ariaLive}
      aria-atomic="true"
    >
      <div
        className={`toast__icon toast__icon--${toast.tone}`}
        aria-hidden="true"
      >
        {getToneIcon(toast.tone)}
      </div>
      <div className="toast__body">
        <div className="toast__eyebrow">{toneLabel}</div>
        <h2 className="toast__title">{toast.title}</h2>
        {toast.description ? (
          <p className="toast__description">{toast.description}</p>
        ) : null}
        {toast.action ? (
          <div className="toast__actions">
            <button
              type="button"
              className="secondary toast__action"
              onClick={() => {
                void handleAction();
              }}
              disabled={isActionPending}
            >
              {isActionPending ? "Working…" : toast.action.label}
            </button>
          </div>
        ) : null}
      </div>
      {toast.dismissible ? (
        <button
          type="button"
          className="secondary toast__dismiss"
          aria-label={`Dismiss ${toneLabel.toLowerCase()} notification`}
          onClick={() => onDismiss(toast.id)}
        >
          Close
        </button>
      ) : null}
    </article>
  );
}
