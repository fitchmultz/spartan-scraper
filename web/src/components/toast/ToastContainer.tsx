/**
 * Purpose: Render the active global toast stack in the correct viewport position for desktop and mobile layouts.
 * Responsibilities: Limit the visible toast set, preserve newest-first ordering, and hand dismissal events down to individual toast cards.
 * Scope: Toast stack layout only.
 * Usage: Rendered internally by `ToastProvider`.
 * Invariants/Assumptions: The newest notifications should remain visible, desktop shows up to four, and mobile CSS further reduces the visible stack without reordering announcements.
 */

import type { ToastRecord } from "./ToastProvider";
import { Toast } from "./Toast";

interface ToastContainerProps {
  toasts: ToastRecord[];
  onDismiss: (id: string) => void;
}

const MAX_VISIBLE_TOASTS = 4;

export function ToastContainer({ toasts, onDismiss }: ToastContainerProps) {
  const visibleToasts = toasts.slice(-MAX_VISIBLE_TOASTS).reverse();

  if (visibleToasts.length === 0) {
    return null;
  }

  return (
    <section className="toast-viewport" aria-label="Notifications">
      {visibleToasts.map((toast) => (
        <Toast
          key={`${toast.id}-${toast.updatedAt}`}
          toast={toast}
          onDismiss={onDismiss}
        />
      ))}
    </section>
  );
}
