/**
 * Purpose: Compose the split template detail, draft-session, and promotion hooks into one route-facing controller.
 * Responsibilities: Bridge library selection, workspace draft actions, assistant state, and promotion seeds into the prop groups consumed by the Templates route controllers.
 * Scope: Templates-route controller composition only; UI rendering lives in `TemplateRouteControllers.tsx`.
 * Usage: Call from `TemplateManager.tsx`, then spread the returned prop groups into the dedicated template controllers.
 * Invariants/Assumptions: The composed controller remains the only route-facing API for the Templates surface, and assistant actions never auto-save without an explicit workspace command.
 */

import { useMemo } from "react";

import type { ComponentStatus, Template } from "../../api";
import type { TemplatePromotionSeed } from "../../types/promotion";
import { isTemplateWorkspaceDraftDirty } from "./templateRouteControllerShared";
import { useTemplateAssistantState } from "./useTemplateAssistantState";
import { useTemplateDraftSessionState } from "./useTemplateDraftPersistence";
import { useTemplateDetailLoader } from "./useTemplateDetailLoader";
import { useTemplateDraftSession } from "./useTemplateDraftSession";
import { useTemplatePromotionState } from "./useTemplatePromotionState";

interface TemplateRouteControllerOptions {
  templateNames: string[];
  onTemplatesChanged: () => void;
  aiStatus?: ComponentStatus | null;
  promotionSeed?: TemplatePromotionSeed | null;
  onClearPromotionSeed?: () => void;
}

export function useTemplateRouteController({
  templateNames,
  onTemplatesChanged,
  aiStatus = null,
  promotionSeed = null,
  onClearPromotionSeed,
}: TemplateRouteControllerOptions) {
  const [
    workspaceDraftSession,
    setWorkspaceDraftSession,
    clearWorkspaceDraftSession,
  ] = useTemplateDraftSessionState();

  const detailLoader = useTemplateDetailLoader({
    templateNames,
    initialSelectedName: workspaceDraftSession?.selectedName ?? null,
    hasInitialDraftSession: workspaceDraftSession !== null,
  });

  const draftSession = useTemplateDraftSession({
    onTemplatesChanged,
    onClearPromotionSeed,
    selectedName: detailLoader.selectedName,
    setSelectedName: detailLoader.setSelectedName,
    setSelectedTemplate: detailLoader.setSelectedTemplate,
    selectedTemplateData: detailLoader.selectedTemplateData,
    selectedIsBuiltIn: detailLoader.selectedIsBuiltIn,
    preventAutoSelect: detailLoader.preventAutoSelect,
    setDetailError: detailLoader.setDetailError,
    workspaceDraftSession,
    setWorkspaceDraftSession,
    clearWorkspaceDraftSession,
  });

  const assistantState = useTemplateAssistantState({
    draftTemplate: draftSession.draftTemplate,
    activeDraftSource: draftSession.activeDraftSource,
    readOnly: draftSession.readOnly,
    setIsBuilderOpen: draftSession.setIsBuilderOpen,
  });

  useTemplatePromotionState({
    promotionSeed,
    confirmReplaceCurrentDraft: draftSession.confirmReplaceCurrentDraft,
    fetchTemplateDetail: detailLoader.fetchTemplateDetail,
    loadDraft: draftSession.loadDraft,
    preventAutoSelect: detailLoader.preventAutoSelect,
    setDetailError: detailLoader.setDetailError,
    setIsLoadingDetail: detailLoader.setIsLoadingDetail,
    setPreviewUrl: assistantState.setPreviewUrl,
    setRailTab: assistantState.setRailTab,
    setSelectedName: detailLoader.setSelectedName,
    setSelectedTemplate: detailLoader.setSelectedTemplate,
  });

  const editorTitle =
    draftSession.activeDraftSource === "create"
      ? "New template"
      : draftSession.activeDraftSource === "duplicate"
        ? `Duplicate of ${draftSession.activeOriginalName ?? "template"}`
        : draftSession.activeDraft.name ||
          detailLoader.selectedTemplateData?.name ||
          "Template workspace";

  return useMemo(
    () => ({
      detailError: detailLoader.detailError,
      isBuilderOpen: draftSession.isBuilderOpen,
      toolbarProps: {
        onStartCreate: () => {
          void draftSession.handleStartCreate();
        },
        onOpenAssistant: () => assistantState.openAssistantMode("generate"),
        onOpenVisualBuilder: () => draftSession.setIsBuilderOpen(true),
      },
      libraryProps: {
        templateNames,
        selectedName: detailLoader.selectedName,
        onSelectTemplate: (name: string) => {
          void draftSession.handleSelectTemplate(name);
        },
      },
      workspaceProps: {
        isBuilderOpen: draftSession.isBuilderOpen,
        draftTemplate: draftSession.draftTemplate,
        activeDraft: draftSession.activeDraft,
        activeDraftSource: draftSession.activeDraftSource,
        selectedTemplate: detailLoader.selectedTemplate,
        readOnly: draftSession.readOnly,
        canDeleteSelectedTemplate: draftSession.canDeleteSelectedTemplate,
        editorTitle,
        promotionSeed,
        hiddenDraft: draftSession.hiddenDraftSession
          ? {
              draft: draftSession.hiddenDraftSession.draft,
              originalName: draftSession.hiddenDraftSession.originalName,
            }
          : null,
        isHiddenDraftDirty: draftSession.hiddenDraftSession
          ? isTemplateWorkspaceDraftDirty(draftSession.hiddenDraftSession)
          : false,
        isLoadingDetail: detailLoader.isLoadingDetail,
        hasWorkspaceDraft: draftSession.hasWorkspaceDraft,
        isDirty: draftSession.isDirty,
        isSaving: draftSession.isSaving,
        saveError: draftSession.saveError,
        saveNotice: draftSession.saveNotice,
        onOpenPreview: () => assistantState.openAssistantMode("preview"),
        onOpenAssistant: () => assistantState.openAssistantMode("debug"),
        onStartDuplicate: () => {
          void draftSession.handleStartDuplicate();
        },
        onDelete: () => {
          void draftSession.handleDelete();
        },
        onOpenVisualBuilder: () => draftSession.setIsBuilderOpen(true),
        onClearPromotionSeed,
        onResumeDraft: draftSession.resumeHiddenDraft,
        onDiscardDraft: () => {
          void draftSession.discardWorkspaceDraft();
        },
        onUpdateDraft: draftSession.updateDraft,
        onUpdateSelector: draftSession.handleUpdateSelector,
        onSave: () => {
          void draftSession.handleSave();
        },
        onReset: draftSession.resetWorkspaceDraft,
        onClose: draftSession.closeWorkspaceDraft,
        onBuilderSave: draftSession.handleBuilderSave,
        onBuilderCancel: () => draftSession.setIsBuilderOpen(false),
      },
      assistantProps: {
        railTab: assistantState.railTab,
        draftTemplate: draftSession.draftTemplate,
        previewUrl: assistantState.previewUrl,
        aiStatus,
        onModeChange: assistantState.setRailTab,
        onPreviewUrlChange: assistantState.setPreviewUrl,
        onApplyTemplate: (template: Template) => {
          void draftSession.handleApplyTemplate(
            template,
            assistantState.applyTemplateSource,
          );
        },
      },
    }),
    [
      aiStatus,
      detailLoader.detailError,
      detailLoader.isLoadingDetail,
      detailLoader.selectedName,
      detailLoader.selectedTemplate,
      draftSession.activeDraft,
      draftSession.activeDraftSource,
      draftSession.canDeleteSelectedTemplate,
      draftSession.closeWorkspaceDraft,
      draftSession.discardWorkspaceDraft,
      draftSession.draftTemplate,
      draftSession.handleApplyTemplate,
      draftSession.handleBuilderSave,
      draftSession.handleDelete,
      draftSession.handleSave,
      draftSession.handleSelectTemplate,
      draftSession.handleStartCreate,
      draftSession.handleStartDuplicate,
      draftSession.handleUpdateSelector,
      draftSession.hasWorkspaceDraft,
      draftSession.hiddenDraftSession,
      draftSession.isBuilderOpen,
      draftSession.isDirty,
      draftSession.isSaving,
      draftSession.readOnly,
      draftSession.resetWorkspaceDraft,
      draftSession.resumeHiddenDraft,
      draftSession.saveError,
      draftSession.saveNotice,
      draftSession.setIsBuilderOpen,
      draftSession.updateDraft,
      editorTitle,
      assistantState,
      onClearPromotionSeed,
      promotionSeed,
      templateNames,
    ],
  );
}
