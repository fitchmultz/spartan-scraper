/**
 * Purpose: Compose template draft persistence and template mutation actions into one route-local workspace hook.
 * Responsibilities: Bridge persisted draft-session state, workspace mutation actions, and controller-facing notice state without letting either concern dominate the file.
 * Scope: Template workspace hook composition only; saved-template detail loading and promotion-seed orchestration stay in sibling hooks.
 * Usage: Called from `useTemplateRouteController()` after the selected template detail has been resolved.
 * Invariants/Assumptions: Persisted draft behavior and mutation behavior remain independently testable, while the route controller keeps consuming one stable workspace-hook API.
 */

import {
  useCallback,
  useMemo,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";

import type { SelectorRule, Template, TemplateDetail } from "../../api";
import { useTemplateDraftPersistence } from "./useTemplateDraftPersistence";
import { useTemplateMutationActions } from "./useTemplateMutationActions";
import type { TemplateWorkspaceDraftSession } from "./templateRouteControllerShared";

interface UseTemplateDraftSessionOptions {
  onTemplatesChanged: () => void;
  onClearPromotionSeed?: () => void;
  selectedName: string | null;
  setSelectedName: (name: string | null) => void;
  setSelectedTemplate: (template: TemplateDetail | null) => void;
  selectedTemplateData: Template | null;
  selectedIsBuiltIn: boolean;
  preventAutoSelect: () => void;
  setDetailError: (error: string | null) => void;
  workspaceDraftSession: TemplateWorkspaceDraftSession | null;
  setWorkspaceDraftSession: Dispatch<
    SetStateAction<TemplateWorkspaceDraftSession | null>
  >;
  clearWorkspaceDraftSession: () => void;
}

export function useTemplateDraftSession({
  onTemplatesChanged,
  onClearPromotionSeed,
  selectedName,
  setSelectedName,
  setSelectedTemplate,
  selectedTemplateData,
  selectedIsBuiltIn,
  preventAutoSelect,
  setDetailError,
  workspaceDraftSession,
  setWorkspaceDraftSession,
  clearWorkspaceDraftSession,
}: UseTemplateDraftSessionOptions) {
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveNotice, setSaveNotice] = useState<string | null>(null);

  const clearWorkspaceNotices = useCallback(() => {
    setSaveError(null);
    setSaveNotice(null);
  }, []);

  const persistence = useTemplateDraftPersistence({
    onClearPromotionSeed,
    selectedName,
    setSelectedName,
    selectedTemplateData,
    selectedIsBuiltIn,
    workspaceDraftSession,
    setWorkspaceDraftSession,
    clearWorkspaceDraftSession,
    onDraftChange: clearWorkspaceNotices,
  });

  const handleUpdateSelector = useCallback(
    (selectorId: string, key: keyof SelectorRule, value: string | boolean) => {
      persistence.updateDraft((current) => ({
        ...current,
        selectors: current.selectors.map((selector) =>
          selector.id === selectorId
            ? {
                ...selector,
                rule: { ...selector.rule, [key]: value },
              }
            : selector,
        ),
      }));
    },
    [persistence.updateDraft],
  );

  const mutations = useTemplateMutationActions({
    onTemplatesChanged,
    onClearPromotionSeed,
    selectedName,
    setSelectedName,
    setSelectedTemplate,
    selectedTemplateData,
    selectedIsBuiltIn,
    preventAutoSelect,
    setDetailError,
    activeDraft: persistence.activeDraft,
    activeDraftSource: persistence.activeDraftSource,
    activeOriginalName: persistence.activeOriginalName,
    confirmReplaceCurrentDraft: persistence.confirmReplaceCurrentDraft,
    loadDraft: persistence.loadDraft,
    clearWorkspaceDraftSession,
    setIsBuilderOpen: persistence.setIsBuilderOpen,
    clearWorkspaceNotices,
    setIsSaving,
    setSaveError,
    setSaveNotice,
  });

  return useMemo(
    () => ({
      ...persistence,
      ...mutations,
      handleUpdateSelector,
      isSaving,
      saveError,
      saveNotice,
    }),
    [
      handleUpdateSelector,
      isSaving,
      mutations,
      persistence,
      saveError,
      saveNotice,
    ],
  );
}
