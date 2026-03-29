/**
 * Purpose: Own draft-session state and workspace actions for the Templates route.
 * Responsibilities: Persist local draft sessions, derive editable workspace state, handle save/delete/create/duplicate/apply flows, and keep unsaved-edit protections consistent.
 * Scope: Template workspace draft state only; saved-template detail loading and promotion-seed orchestration stay in sibling hooks.
 * Usage: Called from `useTemplateRouteController()` after the selected template detail has been resolved.
 * Invariants/Assumptions: Workspace drafts persist in session storage, save/delete flows always go through the API, and built-in templates remain read-only until duplicated.
 */

import {
  useCallback,
  useMemo,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";

import {
  createTemplate,
  deleteTemplate,
  updateTemplate,
  type SelectorRule,
  type Template,
  type TemplateDetail,
} from "../../api";
import { useBeforeUnloadPrompt } from "../../hooks/useBeforeUnloadPrompt";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { deepEqual } from "../../lib/diff-utils";
import { useToast } from "../toast";
import {
  buildDraftFromTemplate,
  buildTemplatePayload,
  buildTemplateSnapshot,
  getDuplicateName,
  type TemplateDraftState,
} from "./templateEditorUtils";
import {
  createTemplateWorkspaceDraftSession,
  isTemplateWorkspaceDraftDirty,
  type DraftSource,
  type TemplateWorkspaceDraftSession,
} from "./templateRouteControllerShared";

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
  const toast = useToast();
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveNotice, setSaveNotice] = useState<string | null>(null);
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

  const clearWorkspaceNotices = useCallback(() => {
    setSaveError(null);
    setSaveNotice(null);
  }, []);

  const confirmReplaceCurrentDraft = useCallback(
    async (options?: { title?: string; reason?: string }) => {
      if (!workspaceDraftSession || !isDirty) {
        return true;
      }

      return toast.confirm({
        title: options?.title ?? "Replace the current template draft?",
        description:
          options?.reason ??
          "This opens another local template draft and discards the edits you have not saved yet. Keep the current draft if you still need it.",
        confirmLabel: "Discard draft",
        cancelLabel: "Keep draft",
        tone: "warning",
      });
    },
    [isDirty, toast, workspaceDraftSession],
  );

  const discardWorkspaceDraft = useCallback(
    async (options?: { title?: string; reason?: string }) => {
      if (!workspaceDraftSession) {
        return true;
      }

      const confirmed = await toast.confirm({
        title: options?.title ?? "Discard the local template draft?",
        description:
          options?.reason ??
          (isDirty
            ? "This removes the in-progress template draft. Your unsaved edits will be lost."
            : "This removes the current local template draft from this tab."),
        confirmLabel: "Discard draft",
        cancelLabel: "Keep draft",
        tone: "warning",
      });
      if (!confirmed) {
        return false;
      }

      clearWorkspaceDraftSession();
      onClearPromotionSeed?.();
      clearWorkspaceNotices();
      setIsBuilderOpen(false);
      return true;
    },
    [
      clearWorkspaceDraftSession,
      clearWorkspaceNotices,
      isDirty,
      onClearPromotionSeed,
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
      clearWorkspaceNotices();
      setIsBuilderOpen(false);
    },
    [clearWorkspaceNotices, setSelectedName, setWorkspaceDraftSession],
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
      clearWorkspaceNotices();
    },
    [
      activeDraft,
      activeDraftSource,
      activeOriginalName,
      clearWorkspaceNotices,
      selectedName,
      setWorkspaceDraftSession,
    ],
  );

  const handleUpdateSelector = useCallback(
    (selectorId: string, key: keyof SelectorRule, value: string | boolean) => {
      updateDraft((current) => ({
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
    [updateDraft],
  );

  const handleSave = useCallback(async () => {
    const { payload, error } = buildTemplatePayload(activeDraft);
    if (!payload || error) {
      const message = error ?? "Failed to build template payload.";
      setSaveError(message);
      toast.show({
        tone: "error",
        title: "Template configuration is invalid",
        description: message,
      });
      return;
    }

    const toastId = toast.show({
      tone: "loading",
      title: payload.name ? `Saving ${payload.name}` : "Saving template",
      description: "Persisting your template changes.",
    });

    setIsSaving(true);
    clearWorkspaceNotices();

    try {
      const shouldUpdateSelected =
        activeDraftSource === "selected" &&
        !selectedIsBuiltIn &&
        !!(activeOriginalName ?? selectedTemplateData?.name);

      if (shouldUpdateSelected) {
        const response = await updateTemplate({
          baseUrl: getApiBaseUrl(),
          path: {
            name: activeOriginalName ?? selectedTemplateData?.name ?? "",
          },
          body: payload,
        });

        if (response.error) {
          throw new Error(
            getApiErrorMessage(response.error, "Failed to save template."),
          );
        }
      } else {
        const response = await createTemplate({
          baseUrl: getApiBaseUrl(),
          body: payload,
        });

        if (response.error) {
          throw new Error(
            getApiErrorMessage(response.error, "Failed to save template."),
          );
        }
      }

      clearWorkspaceDraftSession();
      setSelectedTemplate({
        name: payload.name,
        is_built_in: false,
        template: payload,
      });
      onClearPromotionSeed?.();
      preventAutoSelect();
      setSaveNotice("Template saved.");
      setSelectedName(payload.name);
      onTemplatesChanged();
      toast.update(toastId, {
        tone: "success",
        title: "Template saved",
        description: `${payload.name} is ready to reuse from the template library.`,
      });
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to save template.";
      setSaveError(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to save template",
        description: message,
      });
    } finally {
      setIsSaving(false);
    }
  }, [
    activeDraft,
    activeDraftSource,
    activeOriginalName,
    clearWorkspaceDraftSession,
    clearWorkspaceNotices,
    onClearPromotionSeed,
    onTemplatesChanged,
    preventAutoSelect,
    selectedIsBuiltIn,
    selectedTemplateData?.name,
    setSelectedName,
    setSelectedTemplate,
    toast,
  ]);

  const handleDelete = useCallback(async () => {
    if (!selectedTemplateData?.name || selectedIsBuiltIn) {
      return;
    }

    const confirmed = await toast.confirm({
      title: `Delete ${selectedTemplateData.name}?`,
      description:
        "This removes the saved custom template from the library. Built-in templates remain available.",
      confirmLabel: "Delete template",
      cancelLabel: "Keep template",
      tone: "error",
    });
    if (!confirmed) {
      return;
    }

    const toastId = toast.show({
      tone: "loading",
      title: `Deleting ${selectedTemplateData.name}`,
      description: "Removing the saved template from the library.",
    });

    setDetailError(null);
    try {
      const response = await deleteTemplate({
        baseUrl: getApiBaseUrl(),
        path: { name: selectedTemplateData.name },
      });

      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to delete template."),
        );
      }

      clearWorkspaceDraftSession();
      onTemplatesChanged();
      preventAutoSelect();
      setSelectedName(null);
      setSelectedTemplate(null);
      toast.update(toastId, {
        tone: "success",
        title: "Template deleted",
        description: `${selectedTemplateData.name} has been removed from the library.`,
      });
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to delete template.";
      setDetailError(message);
      toast.update(toastId, {
        tone: "error",
        title: "Failed to delete template",
        description: message,
      });
    }
  }, [
    clearWorkspaceDraftSession,
    onTemplatesChanged,
    preventAutoSelect,
    selectedIsBuiltIn,
    selectedTemplateData?.name,
    setDetailError,
    setSelectedName,
    setSelectedTemplate,
    toast,
  ]);

  const handleStartCreate = useCallback(async () => {
    if (
      !(await confirmReplaceCurrentDraft({
        title: "Replace the current template draft?",
        reason:
          "This starts a new local template draft and discards the edits you have not saved yet. Keep the current draft if you still need it.",
      }))
    ) {
      return;
    }

    onClearPromotionSeed?.();
    preventAutoSelect();
    loadDraft(undefined, "create", { selectedName });
  }, [
    confirmReplaceCurrentDraft,
    loadDraft,
    onClearPromotionSeed,
    preventAutoSelect,
    selectedName,
  ]);

  const handleStartDuplicate = useCallback(async () => {
    if (!selectedTemplateData) {
      return;
    }
    if (
      !(await confirmReplaceCurrentDraft({
        title: "Replace the current template draft?",
        reason:
          "This duplicates another saved template into the workspace and discards the edits you have not saved yet. Keep the current draft if you still need it.",
      }))
    ) {
      return;
    }

    onClearPromotionSeed?.();
    preventAutoSelect();
    loadDraft(
      {
        ...selectedTemplateData,
        name: selectedTemplateData.name
          ? getDuplicateName(selectedTemplateData.name)
          : "",
      },
      "duplicate",
      {
        originalName: selectedTemplateData.name,
        selectedName: selectedTemplateData.name,
      },
    );
  }, [
    confirmReplaceCurrentDraft,
    loadDraft,
    onClearPromotionSeed,
    preventAutoSelect,
    selectedTemplateData,
  ]);

  const handleSelectTemplate = useCallback(
    async (name: string) => {
      if (name === selectedName) {
        return;
      }

      if (
        !(await confirmReplaceCurrentDraft({
          title: "Replace the current template draft?",
          reason:
            "This opens another saved template and discards the edits you have not saved yet. Keep the current draft if you still need it.",
        }))
      ) {
        return;
      }

      clearWorkspaceDraftSession();
      onClearPromotionSeed?.();
      preventAutoSelect();
      setSelectedName(name);
      setIsBuilderOpen(false);
      clearWorkspaceNotices();
    },
    [
      clearWorkspaceDraftSession,
      clearWorkspaceNotices,
      confirmReplaceCurrentDraft,
      onClearPromotionSeed,
      preventAutoSelect,
      selectedName,
      setSelectedName,
    ],
  );

  const handleApplyTemplate = useCallback(
    async (template: Template, source: DraftSource) => {
      if (
        !(await confirmReplaceCurrentDraft({
          title: "Replace the current template draft?",
          reason:
            "This applies another template to the workspace and discards the edits you have not saved yet. Keep the current draft if you still need it.",
        }))
      ) {
        return;
      }

      onClearPromotionSeed?.();
      preventAutoSelect();
      loadDraft(template, source, {
        originalName:
          source === "selected"
            ? (activeOriginalName ??
              selectedTemplateData?.name ??
              template.name ??
              null)
            : source === "duplicate"
              ? (activeOriginalName ?? selectedTemplateData?.name ?? null)
              : null,
        selectedName,
      });
    },
    [
      activeOriginalName,
      confirmReplaceCurrentDraft,
      loadDraft,
      onClearPromotionSeed,
      preventAutoSelect,
      selectedName,
      selectedTemplateData?.name,
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

  const handleBuilderSave = useCallback(
    (template: Template) => {
      setIsBuilderOpen(false);
      clearWorkspaceDraftSession();
      setSelectedTemplate({
        name: template.name,
        is_built_in: false,
        template,
      });
      onClearPromotionSeed?.();
      preventAutoSelect();
      setSelectedName(template.name ?? null);
      clearWorkspaceNotices();
      onTemplatesChanged();
    },
    [
      clearWorkspaceDraftSession,
      clearWorkspaceNotices,
      onClearPromotionSeed,
      onTemplatesChanged,
      preventAutoSelect,
      setSelectedName,
      setSelectedTemplate,
    ],
  );

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
      handleApplyTemplate,
      handleBuilderSave,
      handleDelete,
      handleSave,
      handleSelectTemplate,
      handleStartCreate,
      handleStartDuplicate,
      handleUpdateSelector,
      hasWorkspaceDraft,
      hiddenDraftSession,
      isBuilderOpen,
      isDirty,
      isSaving,
      loadDraft,
      readOnly,
      resetWorkspaceDraft,
      resumeHiddenDraft,
      saveError,
      saveNotice,
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
      handleApplyTemplate,
      handleBuilderSave,
      handleDelete,
      handleSave,
      handleSelectTemplate,
      handleStartCreate,
      handleStartDuplicate,
      handleUpdateSelector,
      hasWorkspaceDraft,
      hiddenDraftSession,
      isBuilderOpen,
      isDirty,
      isSaving,
      loadDraft,
      readOnly,
      resetWorkspaceDraft,
      resumeHiddenDraft,
      saveError,
      saveNotice,
      updateDraft,
    ],
  );
}
