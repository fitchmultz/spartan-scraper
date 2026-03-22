/**
 * Purpose: Provide the inline template management workspace for the Templates route.
 * Responsibilities: Load template inventory and detail, manage create/edit/duplicate/delete flows, and host inline visual-builder, preview, and AI-assisted authoring tools.
 * Scope: Templates route workspace only; app-level routing and shell framing stay outside this component.
 * Usage: Render from the Templates route with the authoritative template name list and refresh callbacks.
 * Invariants/Assumptions: Template detail comes from the API on demand, built-in templates remain non-destructive in place, and workspace actions should preserve draft continuity without modal-first flows.
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
import {
  TemplateAssistantSection,
  type TemplateAssistantMode,
  useAIAssistant,
} from "../ai-assistant";
import { PromotionDraftNotice } from "../promotion/PromotionDraftNotice";
import { ResumableSettingsDraftNotice } from "../settings/ResumableSettingsDraftNotice";
import { useToast } from "../toast";
import { VisualSelectorBuilder } from "../VisualSelectorBuilder";
import { TemplateEditorInline } from "./TemplateEditorInline";
import {
  BUILT_IN_TEMPLATE_NAMES,
  buildDraftFromTemplate,
  buildTemplatePayload,
  buildTemplateSnapshot,
  createSelectorDraft,
  describeTemplate,
  getDuplicateName,
  type TemplateDraftState,
} from "./templateEditorUtils";
import type { TemplatePromotionSeed } from "../../types/promotion";

interface TemplateManagerProps {
  templateNames: string[];
  onTemplatesChanged: () => void;
  aiStatus?: ComponentStatus | null;
  promotionSeed?: TemplatePromotionSeed | null;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
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

export function TemplateManager({
  templateNames,
  onTemplatesChanged,
  aiStatus = null,
  promotionSeed = null,
  onClearPromotionSeed,
  onOpenSourceJob,
}: TemplateManagerProps) {
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

  const handleUpdateSelector = (
    selectorId: string,
    key: keyof SelectorRule,
    value: string | boolean,
  ) => {
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
  };

  const handleSave = async () => {
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
  };

  const handleDelete = async () => {
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
  };

  const handleStartCreate = async () => {
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
  };

  const handleStartDuplicate = async () => {
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
  };

  const handleSelectTemplate = async (name: string) => {
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
  };

  const handleApplyTemplate = async (
    template: Template,
    source: DraftSource,
  ) => {
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
  };

  const closeWorkspaceDraft = useCallback(() => {
    setWorkspaceDraftSession((current) =>
      current ? { ...current, visible: false } : current,
    );
    setIsBuilderOpen(false);
  }, [setWorkspaceDraftSession]);

  const openAssistantMode = (mode: TemplateAssistantMode) => {
    setRailTab(mode);
    setIsBuilderOpen(false);
    aiAssistant.open({
      surface: "templates",
      templateName: draftTemplate.name || undefined,
      templateSnapshot: draftTemplate as Record<string, unknown>,
      selectedUrl: previewUrl || undefined,
    });
  };

  const editorTitle =
    activeDraftSource === "create"
      ? "New template"
      : activeDraftSource === "duplicate"
        ? `Duplicate of ${activeOriginalName ?? "template"}`
        : activeDraft.name ||
          selectedTemplateData?.name ||
          "Template workspace";

  return (
    <div className="template-manager-shell">
      <section className="panel template-manager__toolbar">
        <div className="template-manager__toolbar-copy">
          <h3>Template workspace</h3>
          <p>
            Edit rules inline, preview them against a real page, and use AI
            without losing your place.
          </p>
        </div>

        <div className="template-manager__toolbar-actions">
          <button
            type="button"
            className="btn btn--secondary"
            onClick={handleStartCreate}
          >
            New Template
          </button>
          <button
            type="button"
            className="btn btn--secondary"
            onClick={() => openAssistantMode("generate")}
          >
            Open AI assistant
          </button>
          <button
            type="button"
            className="btn btn--secondary"
            onClick={() => setIsBuilderOpen(true)}
          >
            Open Visual Builder
          </button>
        </div>
      </section>

      <section className="template-manager__workspace">
        <aside className="template-manager__library">
          <div className="template-manager__library-header">
            <h4>Templates</h4>
            <span>{templateNames.length}</span>
          </div>

          {templateNames.length > 0 ? (
            <ul
              className="template-manager__list"
              aria-label="Extraction template list"
            >
              {templateNames.map((name) => {
                const isSelected = name === selectedName;
                const templateKind = BUILT_IN_TEMPLATE_NAMES.includes(
                  name as (typeof BUILT_IN_TEMPLATE_NAMES)[number],
                )
                  ? "Built-in"
                  : "Custom";

                return (
                  <li key={name}>
                    <button
                      type="button"
                      className={`template-manager__list-item ${isSelected ? "is-selected" : ""}`}
                      onClick={() => handleSelectTemplate(name)}
                    >
                      <div className="template-manager__list-item-top">
                        <strong>{name}</strong>
                        <span
                          className={`badge ${templateKind === "Built-in" ? "running" : "success"}`}
                        >
                          {templateKind}
                        </span>
                      </div>
                      <span>Open in workspace</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          ) : null}
        </aside>

        {isBuilderOpen ? (
          <section className="template-manager__builder-surface">
            <VisualSelectorBuilder
              key={`${activeDraft.name || "new"}-${activeDraftSource}`}
              initialTemplate={draftTemplate}
              onSave={(template) => {
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
              }}
              onCancel={() => setIsBuilderOpen(false)}
            />
          </section>
        ) : (
          <>
            <section className="template-manager__editor-surface">
              <div className="template-manager__detail-header">
                <div>
                  <div className="template-manager__detail-eyebrow">
                    <span
                      className={`badge ${readOnly ? "running" : "success"}`}
                    >
                      {readOnly ? "Built-in template" : "Editable workspace"}
                    </span>
                  </div>
                  <h3>{editorTitle}</h3>
                  <p>
                    {selectedTemplate && activeDraftSource === "selected"
                      ? describeTemplate(selectedTemplate)
                      : "Changes stay in the workspace until you explicitly save them."}
                  </p>
                </div>

                <div className="template-manager__detail-actions">
                  <button
                    type="button"
                    className="btn btn--secondary btn--small"
                    onClick={() => openAssistantMode("preview")}
                  >
                    Preview
                  </button>
                  <button
                    type="button"
                    className="btn btn--secondary btn--small"
                    onClick={() => openAssistantMode("debug")}
                  >
                    Open AI assistant
                  </button>
                  {readOnly ? (
                    <button
                      type="button"
                      className="btn btn--secondary btn--small"
                      onClick={handleStartDuplicate}
                    >
                      Duplicate to Edit
                    </button>
                  ) : canDeleteSelectedTemplate ? (
                    <button
                      type="button"
                      className="btn btn--danger btn--small"
                      onClick={handleDelete}
                    >
                      Delete
                    </button>
                  ) : null}
                </div>
              </div>

              {promotionSeed ? (
                <PromotionDraftNotice
                  title="Template draft seeded from a verified job"
                  description="This workspace starts from the reusable extraction structure Spartan could safely recover from the successful source job."
                  seed={promotionSeed}
                  onOpenSourceJob={onOpenSourceJob}
                  onClear={onClearPromotionSeed}
                />
              ) : null}

              {hiddenDraftSession ? (
                <ResumableSettingsDraftNotice
                  title={`Template draft for ${
                    hiddenDraftSession.draft.name ||
                    hiddenDraftSession.originalName ||
                    "the current workspace"
                  }${
                    isTemplateWorkspaceDraftDirty(hiddenDraftSession)
                      ? " has unsaved edits."
                      : " is still available in this tab."
                  }`}
                  description="Close keeps this draft available in the current tab. Resume it when you want to continue editing, or discard it explicitly once you no longer need it."
                  resumeLabel="Resume template draft"
                  discardLabel="Discard template draft"
                  onResume={() =>
                    setWorkspaceDraftSession((current) =>
                      current ? { ...current, visible: true } : current,
                    )
                  }
                  onDiscard={() => {
                    void discardWorkspaceDraft();
                  }}
                />
              ) : null}

              {isLoadingDetail &&
              activeDraftSource === "selected" &&
              !hasWorkspaceDraft ? (
                <div className="template-manager__empty">
                  Loading template details…
                </div>
              ) : null}

              {!hiddenDraftSession ? (
                <TemplateEditorInline
                  draft={activeDraft}
                  readOnly={readOnly}
                  isDirty={isDirty}
                  isSaving={isSaving}
                  error={saveError}
                  notice={saveNotice}
                  onNameChange={(value) =>
                    updateDraft((current) => ({ ...current, name: value }))
                  }
                  onUpdateSelector={handleUpdateSelector}
                  onAddSelector={() =>
                    updateDraft((current) => ({
                      ...current,
                      selectors: [...current.selectors, createSelectorDraft()],
                    }))
                  }
                  onRemoveSelector={(selectorId) =>
                    updateDraft((current) => ({
                      ...current,
                      selectors: current.selectors.filter(
                        (selector) => selector.id !== selectorId,
                      ),
                    }))
                  }
                  onJsonldTextChange={(value) =>
                    updateDraft((current) => ({
                      ...current,
                      jsonldText: value,
                    }))
                  }
                  onRegexTextChange={(value) =>
                    updateDraft((current) => ({ ...current, regexText: value }))
                  }
                  onNormalizeTextChange={(value) =>
                    updateDraft((current) => ({
                      ...current,
                      normalizeText: value,
                    }))
                  }
                  onSave={handleSave}
                  onReset={() => {
                    if (workspaceDraftSession) {
                      if (
                        activeDraftSource === "selected" &&
                        selectedTemplateData
                      ) {
                        loadDraft(selectedTemplateData, "selected", {
                          originalName: selectedTemplateData.name,
                          selectedName,
                        });
                        return;
                      }

                      loadDraft(undefined, "create", { selectedName });
                    }
                  }}
                  onClose={hasWorkspaceDraft ? closeWorkspaceDraft : undefined}
                  onDiscard={
                    hasWorkspaceDraft
                      ? () => {
                          void discardWorkspaceDraft();
                        }
                      : undefined
                  }
                />
              ) : null}
            </section>

            <TemplateAssistantSection
              mode={railTab}
              onModeChange={setRailTab}
              draftTemplate={draftTemplate}
              previewUrl={previewUrl}
              aiStatus={aiStatus}
              onPreviewUrlChange={setPreviewUrl}
              onApplyTemplate={(template) => {
                void handleApplyTemplate(
                  template,
                  railTab === "generate"
                    ? "create"
                    : readOnly
                      ? "duplicate"
                      : activeDraftSource,
                );
              }}
            />
          </>
        )}
      </section>

      {detailError ? <div className="form-error">{detailError}</div> : null}
    </div>
  );
}

export default TemplateManager;
