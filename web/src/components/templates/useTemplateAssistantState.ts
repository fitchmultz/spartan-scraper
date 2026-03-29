/**
 * Purpose: Own template assistant rail state and assistant-launch wiring for the Templates route.
 * Responsibilities: Track assistant mode and preview URL, open the shared AI assistant with the current draft context, and decide how assistant-generated templates should re-enter the workspace.
 * Scope: Template assistant state only; draft persistence, mutations, and promotion handling stay in sibling hooks.
 * Usage: Called from `useTemplateRouteController()` after the current draft state has been resolved.
 * Invariants/Assumptions: Assistant state is route-local, opening the assistant always closes the visual builder, and generated templates re-enter the workspace through an explicit source mode.
 */

import { useCallback, useMemo, useState } from "react";

import type { Template } from "../../api";
import { useAIAssistant, type TemplateAssistantMode } from "../ai-assistant";
import type { DraftSource } from "./templateRouteControllerShared";

interface UseTemplateAssistantStateOptions {
  draftTemplate: Template;
  activeDraftSource: DraftSource;
  readOnly: boolean;
  setIsBuilderOpen: (value: boolean) => void;
}

export function useTemplateAssistantState({
  draftTemplate,
  activeDraftSource,
  readOnly,
  setIsBuilderOpen,
}: UseTemplateAssistantStateOptions) {
  const aiAssistant = useAIAssistant();
  const [railTab, setRailTab] = useState<TemplateAssistantMode>("preview");
  const [previewUrl, setPreviewUrl] = useState("");

  const openAssistantMode = useCallback(
    (mode: TemplateAssistantMode) => {
      setRailTab(mode);
      setIsBuilderOpen(false);
      aiAssistant.open({
        surface: "templates",
        templateName: draftTemplate.name || undefined,
        templateSnapshot: draftTemplate as Record<string, unknown>,
        selectedUrl: previewUrl || undefined,
      });
    },
    [aiAssistant, draftTemplate, previewUrl, setIsBuilderOpen],
  );

  const applyTemplateSource = useMemo<DraftSource>(
    () =>
      railTab === "generate"
        ? "create"
        : readOnly
          ? "duplicate"
          : activeDraftSource,
    [activeDraftSource, railTab, readOnly],
  );

  return useMemo(
    () => ({
      applyTemplateSource,
      openAssistantMode,
      previewUrl,
      railTab,
      setPreviewUrl,
      setRailTab,
    }),
    [applyTemplateSource, openAssistantMode, previewUrl, railTab],
  );
}
