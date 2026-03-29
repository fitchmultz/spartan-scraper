/**
 * Purpose: Own template workspace mutation actions that act on the persisted draft session.
 * Responsibilities: Handle save/delete/create/duplicate/select/apply flows and keep template-library mutations separate from draft persistence state.
 * Scope: Template mutation actions only; persisted draft state and unsaved-draft protections live in `useTemplateDraftPersistence.ts`.
 * Usage: Called from `useTemplateDraftSession()` after draft persistence and selected template detail have been resolved.
 * Invariants/Assumptions: Save/delete flows always go through the API, built-in templates remain non-destructive until duplicated, and mutations clear stale notices before rewriting workspace state.
 */

import { useCallback, useMemo } from "react";

import {
  createTemplate,
  deleteTemplate,
  updateTemplate,
  type Template,
  type TemplateDetail,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { useToast } from "../toast";
import {
  buildTemplatePayload,
  getDuplicateName,
  type TemplateDraftState,
} from "./templateEditorUtils";
import type { DraftSource } from "./templateRouteControllerShared";
import type { TemplateDraftReplacementRequest } from "./templateDraftGuardrails";

interface UseTemplateMutationActionsOptions {
  onTemplatesChanged: () => void;
  onClearPromotionSeed?: () => void;
  selectedName: string | null;
  setSelectedName: (name: string | null) => void;
  setSelectedTemplate: (template: TemplateDetail | null) => void;
  selectedTemplateData: Template | null;
  selectedIsBuiltIn: boolean;
  preventAutoSelect: () => void;
  setDetailError: (error: string | null) => void;
  activeDraft: TemplateDraftState;
  activeDraftSource: DraftSource;
  activeOriginalName: string | null;
  confirmReplaceCurrentDraft: (
    request?: TemplateDraftReplacementRequest,
  ) => Promise<boolean>;
  loadDraft: (
    template: Template | undefined,
    source: DraftSource,
    options?: {
      originalName?: string | null;
      selectedName?: string | null;
      visible?: boolean;
    },
  ) => void;
  clearWorkspaceDraftSession: () => void;
  setIsBuilderOpen: (value: boolean) => void;
  clearWorkspaceNotices: () => void;
  setIsSaving: (value: boolean) => void;
  setSaveError: (value: string | null) => void;
  setSaveNotice: (value: string | null) => void;
}

export function useTemplateMutationActions({
  onTemplatesChanged,
  onClearPromotionSeed,
  selectedName,
  setSelectedName,
  setSelectedTemplate,
  selectedTemplateData,
  selectedIsBuiltIn,
  preventAutoSelect,
  setDetailError,
  activeDraft,
  activeDraftSource,
  activeOriginalName,
  confirmReplaceCurrentDraft,
  loadDraft,
  clearWorkspaceDraftSession,
  setIsBuilderOpen,
  clearWorkspaceNotices,
  setIsSaving,
  setSaveError,
  setSaveNotice,
}: UseTemplateMutationActionsOptions) {
  const toast = useToast();

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
    setIsSaving,
    setSaveError,
    setSaveNotice,
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
    if (!(await confirmReplaceCurrentDraft("create"))) {
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
    if (!(await confirmReplaceCurrentDraft("duplicate"))) {
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

      if (!(await confirmReplaceCurrentDraft("switch-template"))) {
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
      setIsBuilderOpen,
      setSelectedName,
    ],
  );

  const handleApplyTemplate = useCallback(
    async (template: Template, source: DraftSource) => {
      if (!(await confirmReplaceCurrentDraft("apply-template"))) {
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
      setIsBuilderOpen,
      setSelectedName,
      setSelectedTemplate,
    ],
  );

  return useMemo(
    () => ({
      handleApplyTemplate,
      handleBuilderSave,
      handleDelete,
      handleSave,
      handleSelectTemplate,
      handleStartCreate,
      handleStartDuplicate,
    }),
    [
      handleApplyTemplate,
      handleBuilderSave,
      handleDelete,
      handleSave,
      handleSelectTemplate,
      handleStartCreate,
      handleStartDuplicate,
    ],
  );
}
