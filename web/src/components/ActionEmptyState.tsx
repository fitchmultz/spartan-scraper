/**
 * Purpose: Render a reusable action-oriented empty state for product workflows.
 * Responsibilities: Present a concise title, supporting description, optional body content, and one or more next-action buttons.
 * Scope: Shared empty-state presentation only.
 * Usage: Mount inside route or panel components whenever a workspace has no operator data yet.
 * Invariants/Assumptions: Empty states should always suggest an obvious next step and reuse standard button styling.
 */

import type { ReactNode } from "react";

export interface EmptyStateAction {
  label: string;
  onClick: () => void;
  tone?: "primary" | "secondary";
}

interface ActionEmptyStateProps {
  eyebrow?: string;
  title: string;
  description: string;
  actions?: EmptyStateAction[];
  children?: ReactNode;
}

export function ActionEmptyState({
  eyebrow,
  title,
  description,
  actions = [],
  children,
}: ActionEmptyStateProps) {
  return (
    <div className="empty-state">
      {eyebrow ? <div className="empty-state__eyebrow">{eyebrow}</div> : null}
      <h3>{title}</h3>
      <p>{description}</p>

      {children ? <div className="empty-state__body">{children}</div> : null}

      {actions.length > 0 ? (
        <div className="empty-state__actions">
          {actions.map((action) => (
            <button
              key={action.label}
              type="button"
              className={action.tone === "secondary" ? "secondary" : undefined}
              onClick={action.onClick}
            >
              {action.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
