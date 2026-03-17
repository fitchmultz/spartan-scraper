/**
 * Purpose: Provide the inline template management workspace for the Templates route.
 * Responsibilities: Load template inventory and detail, manage create/edit/duplicate/delete flows, and host inline visual-builder, preview, and AI-assisted authoring tools.
 * Scope: Templates route workspace only; app-level routing and shell framing stay outside this component.
 * Usage: Render from the Templates route with the authoritative template name list and refresh callbacks.
 * Invariants/Assumptions: Template detail comes from the API on demand, built-in templates remain non-destructive in place, and workspace actions should preserve draft continuity without modal-first flows.
 */

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  createTemplate,
  deleteTemplate,
  getTemplate,
  updateTemplate,
  type SelectorRule,
  type Template,
  type TemplateDetail,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import {
  TemplateAssistantSection,
  type TemplateAssistantMode,
  useAIAssistant,
} from "../ai-assistant";
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

interface TemplateManagerProps {
  templateNames: string[];
  onTemplatesChanged: () => void;
}

type DraftSource = "selected" | "create" | "duplicate";

export function TemplateManager({
  templateNames,
  onTemplatesChanged,
}: TemplateManagerProps) {
  const aiAssistant = useAIAssistant();
  const toast = useToast();
  const [selectedName, setSelectedName] = useState<string | null>(
    templateNames[0] ?? null,
  );
  const [selectedTemplate, setSelectedTemplate] =
    useState<TemplateDetail | null>(null);
  const [isLoadingDetail, setIsLoadingDetail] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);

  const [draft, setDraft] = useState<TemplateDraftState>(() =>
    buildDraftFromTemplate(),
  );
  const [draftSource, setDraftSource] = useState<DraftSource>(
    templateNames[0] ? "selected" : "create",
  );
  const [originalName, setOriginalName] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveNotice, setSaveNotice] = useState<string | null>(null);
  const [shouldAutoSelectFirst, setShouldAutoSelectFirst] = useState(true);

  const [isBuilderOpen, setIsBuilderOpen] = useState(false);
  const [railTab, setRailTab] = useState<TemplateAssistantMode>("preview");
  const [previewUrl, setPreviewUrl] = useState("");

  const selectedTemplateData = selectedTemplate?.template ?? null;
  const selectedIsBuiltIn =
    selectedTemplate?.is_built_in ??
    (selectedName
      ? BUILT_IN_TEMPLATE_NAMES.includes(
          selectedName as (typeof BUILT_IN_TEMPLATE_NAMES)[number],
        )
      : false);

  const draftTemplate = useMemo(() => buildTemplateSnapshot(draft), [draft]);

  const confirmDiscardChanges = useCallback(async () => {
    if (!isDirty) {
      return true;
    }

    return toast.confirm({
      title: "Discard unsaved template changes?",
      description:
        "Your in-progress edits will be lost and the previous saved state will stay unchanged.",
      confirmLabel: "Discard changes",
      cancelLabel: "Keep editing",
      tone: "warning",
    });
  }, [isDirty, toast]);

  const loadDraft = useCallback(
    (
      template: Template | undefined,
      source: DraftSource,
      nextOriginalName?: string,
    ) => {
      setDraft(buildDraftFromTemplate(template));
      setDraftSource(source);
      setOriginalName(nextOriginalName ?? template?.name ?? null);
      setIsDirty(false);
      setSaveError(null);
      setSaveNotice(null);
      setIsBuilderOpen(false);
    },
    [],
  );

  useEffect(() => {
    if (templateNames.length === 0) {
      setSelectedName(null);
      setSelectedTemplate(null);
      if (draftSource === "selected") {
        loadDraft(undefined, "create");
      }
      return;
    }

    if (!selectedName && shouldAutoSelectFirst) {
      setDraftSource("selected");
      setSelectedName(templateNames[0] ?? null);
      setShouldAutoSelectFirst(false);
    }
  }, [
    draftSource,
    loadDraft,
    selectedName,
    shouldAutoSelectFirst,
    templateNames,
  ]);

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

          if (!isDirty && draftSource === "selected") {
            loadDraft(detail?.template, "selected", detail?.template?.name);
          }
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
  }, [draftSource, isDirty, loadDraft, selectedName]);

  const updateDraft = (
    updater: (current: TemplateDraftState) => TemplateDraftState,
  ) => {
    setDraft((current) => updater(current));
    setIsDirty(true);
    setSaveError(null);
    setSaveNotice(null);
  };

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
    const { payload, error } = buildTemplatePayload(draft);
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
        draftSource === "selected" &&
        !selectedIsBuiltIn &&
        !!(originalName ?? selectedTemplateData?.name);

      if (shouldUpdateSelected) {
        const response = await updateTemplate({
          baseUrl: getApiBaseUrl(),
          path: { name: originalName ?? selectedTemplateData?.name ?? "" },
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

      setSelectedTemplate({
        name: payload.name,
        is_built_in: false,
        template: payload,
      });
      setShouldAutoSelectFirst(false);
      loadDraft(payload, "selected", payload.name);
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

      onTemplatesChanged();
      setShouldAutoSelectFirst(false);
      setSelectedName(null);
      setSelectedTemplate(null);
      loadDraft(undefined, "create");
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
    if (!(await confirmDiscardChanges())) {
      return;
    }

    setShouldAutoSelectFirst(false);
    loadDraft(undefined, "create");
  };

  const handleStartDuplicate = async () => {
    if (!selectedTemplateData || !(await confirmDiscardChanges())) {
      return;
    }

    setShouldAutoSelectFirst(false);
    loadDraft(
      {
        ...selectedTemplateData,
        name: selectedTemplateData.name
          ? getDuplicateName(selectedTemplateData.name)
          : "",
      },
      "duplicate",
      selectedTemplateData.name,
    );
  };

  const handleSelectTemplate = async (name: string) => {
    if (name === selectedName) {
      return;
    }

    if (!(await confirmDiscardChanges())) {
      return;
    }

    setDraftSource("selected");
    setShouldAutoSelectFirst(false);
    setSelectedName(name);
    setIsBuilderOpen(false);
    setSaveError(null);
    setSaveNotice(null);
  };

  const handleApplyTemplate = (template: Template, source: DraftSource) => {
    setShouldAutoSelectFirst(false);
    loadDraft(template, source, template.name);
    setRailTab("preview");
  };

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
    draftSource === "create"
      ? "New template"
      : draftSource === "duplicate"
        ? `Duplicate of ${originalName ?? "template"}`
        : draft.name || selectedTemplateData?.name || "Template workspace";

  const readOnly = draftSource === "selected" && selectedIsBuiltIn;

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

          {templateNames.length === 0 ? (
            <div className="template-manager__empty">
              No saved templates yet. Start a new one, open the visual builder,
              or generate a draft with AI.
            </div>
          ) : (
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
          )}
        </aside>

        {isBuilderOpen ? (
          <section className="template-manager__builder-surface">
            <VisualSelectorBuilder
              key={`${draft.name || "new"}-${draftSource}`}
              initialTemplate={draftTemplate}
              onSave={(template) => {
                setIsBuilderOpen(false);
                setSelectedTemplate({
                  name: template.name,
                  is_built_in: false,
                  template,
                });
                setShouldAutoSelectFirst(false);
                setSelectedName(template.name ?? null);
                loadDraft(template, "selected", template.name);
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
                    {selectedTemplate && draftSource === "selected"
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
                  ) : (
                    selectedTemplateData?.name && (
                      <button
                        type="button"
                        className="btn btn--danger btn--small"
                        onClick={handleDelete}
                      >
                        Delete
                      </button>
                    )
                  )}
                </div>
              </div>

              {isLoadingDetail && draftSource === "selected" ? (
                <div className="template-manager__empty">
                  Loading template details…
                </div>
              ) : null}

              <TemplateEditorInline
                draft={draft}
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
                  updateDraft((current) => ({ ...current, jsonldText: value }))
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
                  if (draftSource === "selected" && selectedTemplateData) {
                    loadDraft(
                      selectedTemplateData,
                      "selected",
                      selectedTemplateData.name,
                    );
                    return;
                  }
                  loadDraft(undefined, "create");
                }}
              />
            </section>

            <TemplateAssistantSection
              mode={railTab}
              onModeChange={setRailTab}
              draftTemplate={draftTemplate}
              previewUrl={previewUrl}
              onPreviewUrlChange={setPreviewUrl}
              onApplyTemplate={(template) => {
                handleApplyTemplate(
                  template,
                  railTab === "generate"
                    ? "create"
                    : readOnly
                      ? "duplicate"
                      : draftSource,
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
