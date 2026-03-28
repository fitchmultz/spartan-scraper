/**
 * Purpose: Own the Templates route state and action orchestration behind a controller-friendly hook.
 * Responsibilities: Load template detail, preserve route-local draft/session continuity, manage create/edit/duplicate/delete flows, and expose stable props for the split library, workspace, and assistant controllers.
 * Scope: Templates-route controller state only; rendered UI lives in `TemplateRouteControllers.tsx`.
 * Usage: Call from `TemplateManager.tsx`, then spread the returned prop groups into the dedicated template controllers.
 * Invariants/Assumptions: The controller hook remains the single source of truth for template-route state, and assistant/builder flows never auto-save without an explicit route action.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  createTemplate,
  deleteTemplate,
  getTemplate,
  updateTemplate,
  type ComponentStatus,
  type SelectorRule,
  type Template,
  type TemplateDetail,
} from "../../api";
import { useBeforeUnloadPrompt } from "../../hooks/useBeforeUnloadPrompt";
import { useSessionStorageState } from "../../hooks/useSessionStorageState";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { deepEqual } from "../../lib/diff-utils";
import { useAIAssistant, type TemplateAssistantMode } from "../ai-assistant";
import { useToast } from "../toast";
import {
  BUILT_IN_TEMPLATE_NAMES,
  buildDraftFromTemplate,
  buildTemplatePayload,
  buildTemplateSnapshot,
  getDuplicateName,
  type TemplateDraftState,
} from "./templateEditorUtils";
import type { TemplatePromotionSeed } from "../../types/promotion";

interface TemplateRouteControllerOptions {
  templateNames: string[];
  onTemplatesChanged: () => void;
  aiStatus?: ComponentStatus | null;
  promotionSeed?: TemplatePromotionSeed | null;
  onClearPromotionSeed?: () => void;
}

type DraftSource = "selected" | "create" | "duplicate";

interface TemplateWorkspaceDraftSession {
  source: DraftSource;
  originalName: string | null;
  selectedName: string | null;
  initialTemplate: Template;
  draft: TemplateDraftState;
  visible: boolean;
}

const TEMPLATE_WORKSPACE_DRAFT_SESSION_KEY =
  "spartan.templates.workspace-draft-session";

function createTemplateWorkspaceDraftSession(
  template: Template | undefined,
  source: DraftSource,
  options?: {
    originalName?: string | null;
    selectedName?: string | null;
    visible?: boolean;
  },
): TemplateWorkspaceDraftSession {
  const draft = buildDraftFromTemplate(template);

  return {
    source,
    originalName:
      options?.originalName !== undefined
        ? options.originalName
        : (template?.name ?? null),
    selectedName:
      options?.selectedName !== undefined
        ? options.selectedName
        : (template?.name ?? null),
    initialTemplate: buildTemplateSnapshot(draft),
    draft,
    visible: options?.visible ?? true,
  };
}

function isTemplateWorkspaceDraftDirty(
  session: TemplateWorkspaceDraftSession,
): boolean {
  return !deepEqual(
    buildTemplateSnapshot(session.draft),
    session.initialTemplate,
  );
}

export function useTemplateRouteController({
  templateNames,
  onTemplatesChanged,
  aiStatus = null,
  promotionSeed = null,
  onClearPromotionSeed,
}: TemplateRouteControllerOptions) {
  const aiAssistant = useAIAssistant();
  const toast = useToast();
  const [
    workspaceDraftSession,
    setWorkspaceDraftSession,
    clearWorkspaceDraftSession,
  ] = useSessionStorageState<TemplateWorkspaceDraftSession | null>(
    TEMPLATE_WORKSPACE_DRAFT_SESSION_KEY,
    null,
  );
  const [selectedName, setSelectedName] = useState<string | null>(
    () => workspaceDraftSession?.selectedName ?? templateNames[0] ?? null,
  );
  const [selectedTemplate, setSelectedTemplate] =
    useState<TemplateDetail | null>(null);
  const [isLoadingDetail, setIsLoadingDetail] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveNotice, setSaveNotice] = useState<string | null>(null);
  const [shouldAutoSelectFirst, setShouldAutoSelectFirst] = useState(
    () => workspaceDraftSession === null,
  );
  const [isBuilderOpen, setIsBuilderOpen] = useState(false);
  const [railTab, setRailTab] = useState<TemplateAssistantMode>("preview");
  const [previewUrl, setPreviewUrl] = useState("");
  const handledPromotionSeedRef = useRef<TemplatePromotionSeed | null>(null);

  const selectedTemplateData = selectedTemplate?.template ?? null;
  const selectedIsBuiltIn =
    selectedTemplate?.is_built_in ??
    (selectedName
      ? BUILT_IN_TEMPLATE_NAMES.includes(
          selectedName as (typeof BUILT_IN_TEMPLATE_NAMES)[number],
        )
      : false);
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
      setSaveError(null);
      setSaveNotice(null);
      setIsBuilderOpen(false);
      return true;
    },
    [
      clearWorkspaceDraftSession,
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
      setSaveError(null);
      setSaveNotice(null);
      setIsBuilderOpen(false);
    },
    [setWorkspaceDraftSession],
  );

  useEffect(() => {
    if (!promotionSeed) {
      handledPromotionSeedRef.current = null;
      return;
    }

    if (handledPromotionSeedRef.current === promotionSeed) {
      return;
    }
    handledPromotionSeedRef.current = promotionSeed;

    let cancelled = false;

    const applyPromotionSeed = async () => {
      if (workspaceDraftSession && isDirty) {
        const confirmed = await toast.confirm({
          title: "Replace the current template draft?",
          description:
            "This verified-job draft will replace the current local template draft. Keep the current draft if you still need those unsaved edits.",
          confirmLabel: "Discard draft",
          cancelLabel: "Keep draft",
          tone: "warning",
        });
        if (!confirmed) {
          return;
        }
      }

      setShouldAutoSelectFirst(false);
      setDetailError(null);
      setPreviewUrl(promotionSeed.previewUrl ?? "");
      setRailTab("preview");

      if (promotionSeed.mode === "inline-template" && promotionSeed.template) {
        setSelectedName(null);
        setSelectedTemplate(null);
        loadDraft(
          {
            ...promotionSeed.template,
            name: promotionSeed.suggestedName,
          },
          "create",
          { selectedName: null },
        );
        return;
      }

      if (
        promotionSeed.mode === "named-template" &&
        promotionSeed.templateName
      ) {
        setIsLoadingDetail(true);
        try {
          const response = await getTemplate({
            baseUrl: getApiBaseUrl(),
            path: { name: promotionSeed.templateName },
          });

          if (response.error) {
            throw new Error(
              getApiErrorMessage(
                response.error,
                "Failed to load template details.",
              ),
            );
          }

          if (cancelled) {
            return;
          }

          const detail = response.data ?? null;
          setSelectedName(detail?.template?.name ?? promotionSeed.templateName);
          setSelectedTemplate(detail);
          loadDraft(
            {
              ...detail?.template,
              name: promotionSeed.suggestedName,
            },
            "duplicate",
            {
              originalName: detail?.template?.name,
              selectedName:
                detail?.template?.name ?? promotionSeed.templateName,
            },
          );
        } catch (error) {
          if (!cancelled) {
            setDetailError(
              error instanceof Error
                ? error.message
                : "Failed to load template details.",
            );
          }
        } finally {
          if (!cancelled) {
            setIsLoadingDetail(false);
          }
        }
        return;
      }

      setSelectedName(null);
      setSelectedTemplate(null);
      loadDraft(
        {
          name: promotionSeed.suggestedName,
        },
        "create",
        { selectedName: null },
      );
    };

    void applyPromotionSeed();

    return () => {
      cancelled = true;
    };
  }, [isDirty, loadDraft, promotionSeed, toast, workspaceDraftSession]);

  useEffect(() => {
    if (templateNames.length === 0) {
      setSelectedName(null);
      setSelectedTemplate(null);
      return;
    }

    if (!selectedName && shouldAutoSelectFirst) {
      setSelectedName(templateNames[0] ?? null);
      setShouldAutoSelectFirst(false);
    }
  }, [selectedName, shouldAutoSelectFirst, templateNames]);

  useEffect(() => {
    if (!selectedName) {
      setSelectedTemplate(null);
      setDetailError(null);
      return;
    }

    let cancelled = false;

    const loadTemplate = async () => {
      setIsLoadingDetail(true);
      setDetailError(null);

      try {
        const response = await getTemplate({
          baseUrl: getApiBaseUrl(),
          path: { name: selectedName },
        });

        if (response.error) {
          throw new Error(
            getApiErrorMessage(
              response.error,
              "Failed to load template details.",
            ),
          );
        }

        if (!cancelled) {
          const detail = response.data ?? null;
          setSelectedTemplate(detail);
          setWorkspaceDraftSession((current) => {
            if (
              !current ||
              current.source !== "selected" ||
              current.originalName !== detail?.template?.name ||
              isTemplateWorkspaceDraftDirty(current)
            ) {
              return current;
            }

            return createTemplateWorkspaceDraftSession(
              detail?.template,
              "selected",
              {
                originalName: detail?.template?.name,
                selectedName,
                visible: current.visible,
              },
            );
          });
        }
      } catch (error) {
        if (!cancelled) {
          setDetailError(
            error instanceof Error
              ? error.message
              : "Failed to load template details.",
          );
          setSelectedTemplate(null);
        }
      } finally {
        if (!cancelled) {
          setIsLoadingDetail(false);
        }
      }
    };

    void loadTemplate();

    return () => {
      cancelled = true;
    };
  }, [selectedName, setWorkspaceDraftSession]);

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
      setSaveError(null);
      setSaveNotice(null);
    },
    [
      activeDraft,
      activeDraftSource,
      activeOriginalName,
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
    setSaveError(null);
    setSaveNotice(null);

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
      setShouldAutoSelectFirst(false);
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
    onClearPromotionSeed,
    onTemplatesChanged,
    selectedIsBuiltIn,
    selectedTemplateData?.name,
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
      setShouldAutoSelectFirst(false);
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
    selectedIsBuiltIn,
    selectedTemplateData?.name,
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
    setShouldAutoSelectFirst(false);
    loadDraft(undefined, "create", { selectedName });
  }, [
    confirmReplaceCurrentDraft,
    loadDraft,
    onClearPromotionSeed,
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
    setShouldAutoSelectFirst(false);
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
      setShouldAutoSelectFirst(false);
      setSelectedName(name);
      setIsBuilderOpen(false);
      setSaveError(null);
      setSaveNotice(null);
    },
    [
      clearWorkspaceDraftSession,
      confirmReplaceCurrentDraft,
      onClearPromotionSeed,
      selectedName,
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
      setShouldAutoSelectFirst(false);
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
      setRailTab("preview");
    },
    [
      activeOriginalName,
      confirmReplaceCurrentDraft,
      loadDraft,
      onClearPromotionSeed,
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
    [aiAssistant, draftTemplate, previewUrl],
  );

  const editorTitle =
    activeDraftSource === "create"
      ? "New template"
      : activeDraftSource === "duplicate"
        ? `Duplicate of ${activeOriginalName ?? "template"}`
        : activeDraft.name ||
          selectedTemplateData?.name ||
          "Template workspace";

  return {
    aiStatus,
    detailError,
    isBuilderOpen,
    toolbarProps: {
      onStartCreate: () => {
        void handleStartCreate();
      },
      onOpenAssistant: () => openAssistantMode("generate"),
      onOpenVisualBuilder: () => setIsBuilderOpen(true),
    },
    libraryProps: {
      templateNames,
      selectedName,
      onSelectTemplate: (name: string) => {
        void handleSelectTemplate(name);
      },
    },
    workspaceProps: {
      isBuilderOpen,
      draftTemplate,
      activeDraft,
      activeDraftSource,
      selectedTemplate,
      readOnly,
      canDeleteSelectedTemplate,
      editorTitle,
      promotionSeed,
      hiddenDraft: hiddenDraftSession
        ? {
            draft: hiddenDraftSession.draft,
            originalName: hiddenDraftSession.originalName,
          }
        : null,
      isHiddenDraftDirty: hiddenDraftSession
        ? isTemplateWorkspaceDraftDirty(hiddenDraftSession)
        : false,
      isLoadingDetail,
      hasWorkspaceDraft,
      isDirty,
      isSaving,
      saveError,
      saveNotice,
      onOpenPreview: () => openAssistantMode("preview"),
      onOpenAssistant: () => openAssistantMode("debug"),
      onStartDuplicate: () => {
        void handleStartDuplicate();
      },
      onDelete: () => {
        void handleDelete();
      },
      onClearPromotionSeed,
      onResumeDraft: () =>
        setWorkspaceDraftSession((current) =>
          current ? { ...current, visible: true } : current,
        ),
      onDiscardDraft: () => {
        void discardWorkspaceDraft();
      },
      onUpdateDraft: updateDraft,
      onUpdateSelector: handleUpdateSelector,
      onSave: () => {
        void handleSave();
      },
      onReset: () => {
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
      },
      onClose: closeWorkspaceDraft,
      onBuilderSave: (template: Template) => {
        setIsBuilderOpen(false);
        clearWorkspaceDraftSession();
        setSelectedTemplate({
          name: template.name,
          is_built_in: false,
          template,
        });
        onClearPromotionSeed?.();
        setShouldAutoSelectFirst(false);
        setSelectedName(template.name ?? null);
        setSaveError(null);
        setSaveNotice(null);
        onTemplatesChanged();
      },
      onBuilderCancel: () => setIsBuilderOpen(false),
    },
    assistantProps: {
      railTab,
      draftTemplate,
      previewUrl,
      aiStatus,
      onModeChange: setRailTab,
      onPreviewUrlChange: setPreviewUrl,
      onApplyTemplate: (template: Template) => {
        void handleApplyTemplate(
          template,
          railTab === "generate"
            ? "create"
            : readOnly
              ? "duplicate"
              : activeDraftSource,
        );
      },
    },
  };
}
