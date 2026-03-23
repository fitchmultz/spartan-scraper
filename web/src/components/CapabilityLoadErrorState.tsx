/**
 * Purpose: Render a panel-local load-failure recovery state for capability surfaces.
 * Responsibilities: Show operator-safe recovery copy, keep the latest error visible, and reuse the shared capability action list and refresh affordance.
 * Scope: Shared web capability load-error presentation only.
 * Usage: Mount from capability panels when initial status loading fails and panel-specific actions should remain available.
 * Invariants/Assumptions: Recovery stays scoped to the current panel, callers provide user-safe copy, and actions remain optional.
 */

import type { ReactNode } from "react";
import type { RecommendedAction } from "../api";
import { ActionEmptyState } from "./ActionEmptyState";
import { CapabilityActionList } from "./CapabilityActionList";

interface CapabilityLoadErrorStateProps {
  eyebrow: string;
  title: string;
  description: string;
  error: string;
  actions: RecommendedAction[];
  onNavigate: (path: string) => void;
  onRefresh: () => Promise<unknown> | undefined;
  refreshLabel?: string;
  children?: ReactNode;
}

export function CapabilityLoadErrorState({
  eyebrow,
  title,
  description,
  error,
  actions,
  onNavigate,
  onRefresh,
  refreshLabel = "Refresh status",
  children,
}: CapabilityLoadErrorStateProps) {
  return (
    <ActionEmptyState
      eyebrow={eyebrow}
      title={title}
      description={description}
      actions={[
        {
          label: refreshLabel,
          onClick: () => {
            void onRefresh();
          },
          tone: "secondary",
        },
      ]}
    >
      {children}

      <div className="system-status__hint">
        <strong>Latest error</strong>
        <span>{error}</span>
      </div>

      <CapabilityActionList
        actions={actions}
        onNavigate={onNavigate}
        onRefresh={onRefresh}
      />
    </ActionEmptyState>
  );
}
