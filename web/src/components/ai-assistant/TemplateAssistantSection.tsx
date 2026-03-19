/**
 * Purpose: Mount the route-aware assistant rail inside the template workspace and keep shared assistant context aligned with the current draft.
 * Responsibilities: Route preview, generation, and debugging through the shared assistant shell while preserving explicit apply actions back into the template editor workspace.
 * Scope: `/templates` assistant behavior only.
 * Usage: Render from `TemplateManager` as the workspace assistant rail.
 * Invariants/Assumptions: Generated or debugged templates are only applied through explicit operator action, and the template workspace remains the source of truth for saved edits.
 */

import { useEffect, useMemo } from "react";
import type { ComponentStatus, Template } from "../../api";
import { TemplatePreviewPane } from "../templates/TemplatePreviewPane";
import { TemplateAssistantWorkspace } from "../templates/TemplateAssistantWorkspace";
import type { AssistantContext } from "./AIAssistantProvider";
import { AIAssistantPanel } from "./AIAssistantPanel";
import { useAIAssistant } from "./useAIAssistant";

export type TemplateAssistantMode = "preview" | "generate" | "debug";

interface TemplateAssistantSectionProps {
  mode: TemplateAssistantMode;
  onModeChange: (mode: TemplateAssistantMode) => void;
  draftTemplate: Template;
  previewUrl: string;
  onPreviewUrlChange: (value: string) => void;
  onApplyTemplate: (template: Template) => void;
  aiStatus?: ComponentStatus | null;
}

export function TemplateAssistantSection({
  mode,
  onModeChange,
  draftTemplate,
  previewUrl,
  onPreviewUrlChange,
  onApplyTemplate,
  aiStatus = null,
}: TemplateAssistantSectionProps) {
  const { setContext } = useAIAssistant();

  const assistantContext = useMemo<AssistantContext>(
    () => ({
      surface: "templates",
      templateName: draftTemplate.name || undefined,
      templateSnapshot: draftTemplate as Record<string, unknown>,
      selectedUrl: previewUrl || undefined,
    }),
    [draftTemplate, previewUrl],
  );

  useEffect(() => {
    setContext(assistantContext);
  }, [assistantContext, setContext]);

  return (
    <AIAssistantPanel
      title="Template assistant"
      routeLabel="/templates"
      aiStatus={mode === "preview" ? null : aiStatus}
      aiManualFallback="Edit the template manually in the main workspace."
      suggestedActions={
        <>
          <button
            type="button"
            className={mode === "preview" ? "active" : "secondary"}
            onClick={() => onModeChange("preview")}
          >
            Preview
          </button>
          <button
            type="button"
            className={mode === "generate" ? "active" : "secondary"}
            onClick={() => onModeChange("generate")}
          >
            Generate
          </button>
          <button
            type="button"
            className={mode === "debug" ? "active" : "secondary"}
            onClick={() => onModeChange("debug")}
          >
            Debug
          </button>
        </>
      }
    >
      {mode === "preview" ? (
        <TemplatePreviewPane
          template={draftTemplate}
          url={previewUrl}
          onUrlChange={onPreviewUrlChange}
        />
      ) : (
        <TemplateAssistantWorkspace
          mode={mode === "generate" ? "generate" : "debug"}
          draftTemplate={draftTemplate}
          url={previewUrl}
          aiStatus={aiStatus}
          onUrlChange={onPreviewUrlChange}
          onApplyTemplate={onApplyTemplate}
        />
      )}
    </AIAssistantPanel>
  );
}
