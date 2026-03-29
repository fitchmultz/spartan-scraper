/**
 * Purpose: Render the shared Settings authoring shell used by native and AI-backed editors.
 * Responsibilities: Present the common Settings header, action bar, draft notices, AI availability notice, JSON inventory toggle, and conditional draft/empty/list regions.
 * Scope: Shared Settings authoring layout only; editor-specific form fields, inventory rows, and AI modal contents are injected by callers.
 * Usage: Wrap render-profile and pipeline-JS Settings editors after shared controller state has been prepared.
 * Invariants/Assumptions: The shell keeps the existing Settings action layout, draft notices use the shared workspace-draft copy, and visible drafts render above the empty state and saved inventory list.
 */

import type { ReactNode } from "react";

import { AIUnavailableNotice } from "../ai-assistant";
import { ResumableSettingsDraftNotice } from "./ResumableSettingsDraftNotice";
import {
  describeHiddenSettingsDraft,
  type SettingsWorkspaceDraftSession,
} from "./workspaceDrafts";

interface HiddenDraftNoticeProps {
  session: Pick<
    SettingsWorkspaceDraftSession<string, unknown, unknown>,
    "attemptId" | "originalName" | "source"
  >;
  isDirty: boolean;
  nativeDraftLabel: string;
  onResume: () => void;
  onDiscard: () => void;
}

interface SettingsAuthoringShellProps {
  loading: boolean;
  loadingLabel: string;
  title: string;
  description: string;
  showJson: boolean;
  onToggleJson: () => void;
  createLabel: string;
  onCreate: () => void;
  onOpenGenerator: () => void;
  aiUnavailable: boolean;
  aiUnavailableMessage?: string | null;
  error?: string | null;
  hiddenDraftNotice?: HiddenDraftNoticeProps | null;
  draftPanel?: ReactNode;
  emptyState?: ReactNode;
  jsonPanel?: ReactNode;
  aiPanels?: ReactNode;
  children: ReactNode;
}

export function SettingsAuthoringShell({
  loading,
  loadingLabel,
  title,
  description,
  showJson,
  onToggleJson,
  createLabel,
  onCreate,
  onOpenGenerator,
  aiUnavailable,
  aiUnavailableMessage,
  error,
  hiddenDraftNotice,
  draftPanel,
  emptyState,
  jsonPanel,
  aiPanels,
  children,
}: SettingsAuthoringShellProps) {
  if (loading) {
    return <div className="p-4 text-center">{loadingLabel}</div>;
  }

  return (
    <div className="space-y-4">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          gap: 12,
          flexWrap: "wrap",
        }}
      >
        <div>
          <h2 style={{ marginBottom: 4 }}>{title}</h2>
          <p style={{ margin: 0, opacity: 0.8 }}>{description}</p>
        </div>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <button type="button" onClick={onToggleJson} className="secondary">
            {showJson ? "Hide JSON" : "Show JSON"}
          </button>
          <button
            type="button"
            onClick={onOpenGenerator}
            disabled={aiUnavailable}
            title={aiUnavailableMessage ?? undefined}
            className={aiUnavailable ? "secondary" : undefined}
          >
            Generate with AI
          </button>
          <button type="button" onClick={onCreate}>
            {createLabel}
          </button>
        </div>
      </div>

      {aiUnavailableMessage ? (
        <AIUnavailableNotice message={aiUnavailableMessage} />
      ) : null}

      {error ? (
        <div className="error" role="alert">
          {error}
        </div>
      ) : null}

      {hiddenDraftNotice ? (
        <ResumableSettingsDraftNotice
          {...describeHiddenSettingsDraft(hiddenDraftNotice.session, {
            isDirty: hiddenDraftNotice.isDirty,
            nativeDraftLabel: hiddenDraftNotice.nativeDraftLabel,
          })}
          onResume={hiddenDraftNotice.onResume}
          onDiscard={hiddenDraftNotice.onDiscard}
        />
      ) : null}

      {draftPanel ?? null}

      {emptyState ?? null}

      {jsonPanel ?? null}

      {aiPanels ?? null}

      <div className="space-y-2">{children}</div>
    </div>
  );
}
