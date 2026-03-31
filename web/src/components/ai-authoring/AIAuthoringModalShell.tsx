/**
 * Purpose: Render the shared modal chrome for AI authoring generator and debugger flows.
 * Responsibilities: Provide the modal overlay, header, capability notice, session-draft notice, and consistent body/footer framing.
 * Scope: Shared presentation shell only; artifact-specific form fields and actions are supplied by callers.
 * Usage: Wrap AI authoring modal content and pass the route-specific body markup plus footer actions.
 * Invariants/Assumptions: Clicks on the overlay close the modal, clicks inside the content do not bubble, and session notices only appear when the caller has an active draft to preserve.
 */

import type { ReactNode } from "react";
import { AIUnavailableNotice } from "../ai-assistant";

interface AIAuthoringModalShellProps {
  title: string;
  titleIcon: string;
  onClose: () => void;
  aiUnavailableMessage?: string | null;
  sessionNotice?: string | null;
  children: ReactNode;
  footer: ReactNode;
}

export function AIAuthoringModalShell({
  title,
  titleIcon,
  onClose,
  aiUnavailableMessage,
  sessionNotice,
  children,
  footer,
}: AIAuthoringModalShellProps) {
  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled via close controls
    <div className="modal-overlay" onClick={onClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: modal content container */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="mr-2 text-purple-400">{titleIcon}</span>
            {title}
          </h2>
          <button
            type="button"
            className="modal-close"
            onClick={onClose}
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="modal-body space-y-4">
          {aiUnavailableMessage ? (
            <AIUnavailableNotice message={aiUnavailableMessage} />
          ) : null}

          {sessionNotice ? (
            <div className="rounded-md border border-sky-500/30 bg-sky-500/10 px-3 py-2 text-sm text-sky-100">
              {sessionNotice}
            </div>
          ) : null}

          {children}
          {footer}
        </div>
      </div>
    </div>
  );
}
