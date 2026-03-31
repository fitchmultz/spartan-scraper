/**
 * Purpose: Own the persisted template workspace draft state and derived draft-session behavior.
 * Responsibilities: Own the session-storage draft state, derive the active template draft, refresh clean selected drafts from authoritative template detail, track builder visibility, protect unsaved edits, and expose draft-state mutations without coupling them to save/delete flows.
 * Scope: Template draft persistence and local workspace state only; save/delete/create/duplicate/apply actions stay in `useTemplateMutationActions.ts`.
 * Usage: `useTemplateRouteController()` reads session state from this module, and `useTemplateDraftSession()` composes the derived draft behavior.
 * Invariants/Assumptions: Session storage is the source of truth for unsaved template drafts, built-in templates stay read-only until duplicated, and only clean selected drafts refresh from authoritative template detail.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";

import type { Template } from "../../api";
import { useBeforeUnloadPrompt } from "../../hooks/useBeforeUnloadPrompt";
import { useSessionStorageState } from "../../hooks/useSessionStorageState";
import { deepEqual } from "../../lib/diff-utils";
import { useToast } from "../toast";
import {
  buildDraftFromTemplate,
  buildTemplateSnapshot,
  type DraftSource,
  type TemplateDraftState,
  type TemplateWorkspaceDraftSession,
  createTemplateWorkspaceDraftSession,
  isTemplateWorkspaceDraftDirty,
} from "./templateRouteControllerShared";
import {
  resolveTemplateDraftDiscardPrompt,
  resolveTemplateDraftReplacementPrompt,
  type TemplateDraftPromptOverride,
  type TemplateDraftReplacementRequest,
} from "./templateDraftGuardrails";

const TEMPLATE_WORKSPACE_DRAFT_SESSION_KEY =
  "spartan.templates.workspace-draft-session";

export function useTemplateDraftSessionState() {
  return useSessionStorageState<TemplateWorkspaceDraftSession | null>(
    TEMPLATE_WORKSPACE_DRAFT_SESSION_KEY,
    null,
  );
}

interface UseTemplateDraftPersistenceOptions {
  onClearPromotionSeed?: () => void;
  selectedName: string | null;
  setSelectedName: (name: string | null) => void;
  selectedTemplateData: Template | null;
  selectedIsBuiltIn: boolean;
  workspaceDraftSession: TemplateWorkspaceDraftSession | null;
  setWorkspaceDraftSession: Dispatch<
    SetStateAction<TemplateWorkspaceDraftSession | null>
  >;
  clearWorkspaceDraftSession: () => void;
  onDraftChange?: () => void;
}

export function useTemplateDraftPersistence({
  onClearPromotionSeed,
  selectedName,
  setSelectedName,
  selectedTemplateData,
  selectedIsBuiltIn,
  workspaceDraftSession,
  setWorkspaceDraftSession,
  clearWorkspaceDraftSession,
  onDraftChange,
}: UseTemplateDraftPersistenceOptions) {
  const toast = useToast();
  const [isBuilderOpen, setIsBuilderOpen] = useState(false);

  const selectedTemplateDraft = useMemo(
    () => buildDraftFromTemplate(selectedTemplateData ?? undefined),
    [selectedTemplateData],
  );
  const activeDraft = workspaceDraftSession?.draft ?? selectedTemplateDraft;
  const activeDraftSource =
    workspaceDraftSession?.source ?? (selectedName ? "selected" : "create");
  const activeOriginalName =
    workspaceDraftSession?.originalName ?? selectedTemplateData?.name ?? null;
  const hasWorkspaceDraft = workspaceDraftSession !== null;
  const readOnly = activeDraftSource === "selected" && selectedIsBuiltIn;
  const hiddenDraftSession =
    workspaceDraftSession && !workspaceDraftSession.visible
      ? workspaceDraftSession
      : null;
  const isDirty = workspaceDraftSession
    ? isTemplateWorkspaceDraftDirty(workspaceDraftSession)
    : false;
  const canDeleteSelectedTemplate =
    activeDraftSource === "selected" &&
    !readOnly &&
    !!selectedTemplateData?.name;
  const draftTemplate = useMemo(
    () => buildTemplateSnapshot(activeDraft),
    [activeDraft],
  );

  useBeforeUnloadPrompt(isDirty);

  useEffect(() => {
    if (!selectedTemplateData || !selectedName) {
      return;
    }

    setWorkspaceDraftSession((current) => {
      if (
        !current ||
        current.source !== "selected" ||
        current.originalName !== selectedTemplateData.name ||
        isTemplateWorkspaceDraftDirty(current)
      ) {
        return current;
      }

      return createTemplateWorkspaceDraftSession(
        selectedTemplateData,
        "selected",
        {
          originalName: selectedTemplateData.name,
          selectedName,
          visible: current.visible,
        },
      );
    });
  }, [selectedName, selectedTemplateData, setWorkspaceDraftSession]);

  const confirmReplaceCurrentDraft = useCallback(
    async (request?: TemplateDraftReplacementRequest) => {
      if (!workspaceDraftSession || !isDirty) {
        return true;
      }

      const prompt = resolveTemplateDraftReplacementPrompt(request);
      return toast.confirm({
        title: prompt.title,
        description: prompt.description,
        confirmLabel: "Discard draft",
        cancelLabel: "Keep draft",
        tone: "warning",
      });
    },
    [isDirty, toast, workspaceDraftSession],
  );

  const discardWorkspaceDraft = useCallback(
    async (options?: TemplateDraftPromptOverride) => {
      if (!workspaceDraftSession) {
        return true;
      }

      const prompt = resolveTemplateDraftDiscardPrompt(isDirty, options);
      const confirmed = await toast.confirm({
        title: prompt.title,
        description: prompt.description,
        confirmLabel: "Discard draft",
        cancelLabel: "Keep draft",
        tone: "warning",
      });
      if (!confirmed) {
        return false;
      }

      clearWorkspaceDraftSession();
      onClearPromotionSeed?.();
      onDraftChange?.();
      setIsBuilderOpen(false);
      return true;
    },
    [
      clearWorkspaceDraftSession,
      isDirty,
      onClearPromotionSeed,
      onDraftChange,
      toast,
      workspaceDraftSession,
    ],
  );

  const loadDraft = useCallback(
    (
      template: Template | undefined,
      source: DraftSource,
      options?: {
        originalName?: string | null;
        selectedName?: string | null;
        visible?: boolean;
      },
    ) => {
      const nextSelectedName =
        options?.selectedName !== undefined
          ? options.selectedName
          : (template?.name ?? null);
      setWorkspaceDraftSession(
        createTemplateWorkspaceDraftSession(template, source, options),
      );
      setSelectedName(nextSelectedName);
      onDraftChange?.();
      setIsBuilderOpen(false);
    },
    [onDraftChange, setSelectedName, setWorkspaceDraftSession],
  );

  const updateDraft = useCallback(
    (updater: (current: TemplateDraftState) => TemplateDraftState) => {
      setWorkspaceDraftSession((current) => {
        if (current) {
          const nextDraft = updater(current.draft);
          return deepEqual(current.draft, nextDraft)
            ? current
            : { ...current, draft: nextDraft, visible: true };
        }

        const nextDraft = updater(activeDraft);
        return {
          source: activeDraftSource,
          originalName: activeOriginalName,
          selectedName,
          initialTemplate: buildTemplateSnapshot(activeDraft),
          draft: nextDraft,
          visible: true,
        };
      });
      onDraftChange?.();
    },
    [
      activeDraft,
      activeDraftSource,
      activeOriginalName,
      onDraftChange,
      selectedName,
      setWorkspaceDraftSession,
    ],
  );

  const closeWorkspaceDraft = useCallback(() => {
    setWorkspaceDraftSession((current) =>
      current ? { ...current, visible: false } : current,
    );
    setIsBuilderOpen(false);
  }, [setWorkspaceDraftSession]);

  const resumeHiddenDraft = useCallback(() => {
    setWorkspaceDraftSession((current) =>
      current ? { ...current, visible: true } : current,
    );
  }, [setWorkspaceDraftSession]);

  const resetWorkspaceDraft = useCallback(() => {
    if (!workspaceDraftSession) {
      return;
    }

    if (activeDraftSource === "selected" && selectedTemplateData) {
      loadDraft(selectedTemplateData, "selected", {
        originalName: selectedTemplateData.name,
        selectedName,
      });
      return;
    }

    loadDraft(undefined, "create", { selectedName });
  }, [
    activeDraftSource,
    loadDraft,
    selectedName,
    selectedTemplateData,
    workspaceDraftSession,
  ]);

  return useMemo(
    () => ({
      activeDraft,
      activeDraftSource,
      activeOriginalName,
      canDeleteSelectedTemplate,
      closeWorkspaceDraft,
      confirmReplaceCurrentDraft,
      discardWorkspaceDraft,
      draftTemplate,
      hasWorkspaceDraft,
      hiddenDraftSession,
      isBuilderOpen,
      isDirty,
      loadDraft,
      readOnly,
      resetWorkspaceDraft,
      resumeHiddenDraft,
      setIsBuilderOpen,
      updateDraft,
    }),
    [
      activeDraft,
      activeDraftSource,
      activeOriginalName,
      canDeleteSelectedTemplate,
      closeWorkspaceDraft,
      confirmReplaceCurrentDraft,
      discardWorkspaceDraft,
      draftTemplate,
      hasWorkspaceDraft,
      hiddenDraftSession,
      isBuilderOpen,
      isDirty,
      loadDraft,
      readOnly,
      resetWorkspaceDraft,
      resumeHiddenDraft,
      updateDraft,
    ],
  );
}
