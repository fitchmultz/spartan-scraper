import { useEffect, useMemo, useState } from "react";

import {
  createTemplate,
  deleteTemplate,
  getTemplate,
  updateTemplate,
  type CreateTemplateRequest,
  type JsonldRule,
  type NormalizeSpec,
  type RegexRule,
  type SelectorRule,
  type Template,
  type TemplateDetail,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { AITemplateDebugger } from "../AITemplateDebugger";
import { VisualSelectorBuilder } from "../VisualSelectorBuilder";

interface TemplateManagerProps {
  templateNames: string[];
  onTemplatesChanged: () => void;
  onOpenAIPreview: () => void;
  onOpenAIGenerator: () => void;
}

interface TemplateEditorModalProps {
  mode: "create" | "edit" | "duplicate";
  originalName?: string;
  initialTemplate?: Template;
  onClose: () => void;
  onSaved: (name: string) => void;
}

interface JsonParseResult<T> {
  data?: T;
  error?: string;
}

interface SelectorDraft {
  id: string;
  rule: SelectorRule;
}

const BUILT_IN_TEMPLATE_NAMES = ["article", "default", "product"] as const;

const EMPTY_SELECTOR: SelectorRule = {
  name: "",
  selector: "",
  attr: "text",
  trim: true,
  all: false,
  required: false,
};

function createDraftId() {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `selector-${Math.random().toString(36).slice(2, 10)}`;
}

function createSelectorDraft(rule?: SelectorRule): SelectorDraft {
  return {
    id: createDraftId(),
    rule: {
      ...EMPTY_SELECTOR,
      ...rule,
    },
  };
}

function getDuplicateName(name: string) {
  return `${name}-copy`;
}

function describeTemplate(detail: TemplateDetail | null) {
  const selectors = detail?.template?.selectors?.length ?? 0;
  const jsonld = detail?.template?.jsonld?.length ?? 0;
  const regex = detail?.template?.regex?.length ?? 0;
  return `${selectors} selector${selectors === 1 ? "" : "s"} · ${jsonld} JSON-LD · ${regex} regex`;
}

function parseJSONInput<T>(label: string, value: string): JsonParseResult<T> {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }

  try {
    return { data: JSON.parse(trimmed) as T };
  } catch (error) {
    return {
      error: `${label} must be valid JSON: ${error instanceof Error ? error.message : "Invalid JSON"}`,
    };
  }
}

function formatJSON(value: unknown) {
  return value ? JSON.stringify(value, null, 2) : "";
}

function normalizeSelectors(selectors: SelectorRule[]) {
  return selectors
    .map((selector) => ({
      ...selector,
      name: selector.name?.trim(),
      selector: selector.selector?.trim(),
      attr: selector.attr?.trim() || "text",
      join: selector.join?.trim() || undefined,
    }))
    .filter((selector) => (selector.name?.length ?? 0) > 0);
}

function ruleKey(rule: SelectorRule) {
  return `${rule.name ?? "selector"}-${rule.selector ?? ""}-${rule.attr ?? "text"}`;
}

function TemplateEditorModal({
  mode,
  originalName,
  initialTemplate,
  onClose,
  onSaved,
}: TemplateEditorModalProps) {
  const [name, setName] = useState(
    initialTemplate?.name ??
      (mode === "duplicate" && originalName
        ? getDuplicateName(originalName)
        : ""),
  );
  const [selectors, setSelectors] = useState<SelectorDraft[]>(
    initialTemplate?.selectors?.length
      ? initialTemplate.selectors.map((selector) =>
          createSelectorDraft(selector),
        )
      : [createSelectorDraft()],
  );
  const [jsonldText, setJsonldText] = useState(
    formatJSON(initialTemplate?.jsonld),
  );
  const [regexText, setRegexText] = useState(
    formatJSON(initialTemplate?.regex),
  );
  const [normalizeText, setNormalizeText] = useState(
    formatJSON(initialTemplate?.normalize),
  );
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const title =
    mode === "edit"
      ? "Edit Template"
      : mode === "duplicate"
        ? "Duplicate Template"
        : "Create Template";

  const saveLabel =
    mode === "edit"
      ? "Save Changes"
      : mode === "duplicate"
        ? "Create Duplicate"
        : "Create Template";

  const updateSelectorField = (
    selectorId: string,
    key: keyof SelectorRule,
    value: string | boolean | number,
  ) => {
    setSelectors((current) =>
      current.map((selector) =>
        selector.id === selectorId
          ? {
              ...selector,
              rule: { ...selector.rule, [key]: value },
            }
          : selector,
      ),
    );
  };

  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setError("Template name is required.");
      return;
    }

    const normalizedSelectors = normalizeSelectors(
      selectors.map((selector) => selector.rule),
    );
    if (normalizedSelectors.length === 0) {
      setError("Add at least one selector rule before saving.");
      return;
    }

    if (
      normalizedSelectors.some(
        (selector) =>
          !selector.selector || selector.selector.trim().length === 0,
      )
    ) {
      setError("Each selector rule needs a CSS selector.");
      return;
    }

    const jsonldResult = parseJSONInput<JsonldRule[]>(
      "JSON-LD rules",
      jsonldText,
    );
    if (jsonldResult.error) {
      setError(jsonldResult.error);
      return;
    }
    if (jsonldResult.data && !Array.isArray(jsonldResult.data)) {
      setError("JSON-LD rules must be a JSON array.");
      return;
    }

    const regexResult = parseJSONInput<RegexRule[]>("Regex rules", regexText);
    if (regexResult.error) {
      setError(regexResult.error);
      return;
    }
    if (regexResult.data && !Array.isArray(regexResult.data)) {
      setError("Regex rules must be a JSON array.");
      return;
    }

    const normalizeResult = parseJSONInput<NormalizeSpec>(
      "Normalization settings",
      normalizeText,
    );
    if (normalizeResult.error) {
      setError(normalizeResult.error);
      return;
    }
    if (
      normalizeResult.data &&
      (Array.isArray(normalizeResult.data) ||
        typeof normalizeResult.data !== "object")
    ) {
      setError("Normalization settings must be a JSON object.");
      return;
    }

    const payload: CreateTemplateRequest = {
      name: trimmedName,
      selectors: normalizedSelectors,
      ...(jsonldResult.data ? { jsonld: jsonldResult.data } : {}),
      ...(regexResult.data ? { regex: regexResult.data } : {}),
      ...(normalizeResult.data ? { normalize: normalizeResult.data } : {}),
    };

    setIsSaving(true);
    setError(null);

    try {
      if (mode === "edit" && originalName) {
        const response = await updateTemplate({
          baseUrl: getApiBaseUrl(),
          path: { name: originalName },
          body: payload,
        });
        if (response.error) {
          throw new Error(String(response.error));
        }
      } else {
        const response = await createTemplate({
          baseUrl: getApiBaseUrl(),
          body: payload,
        });
        if (response.error) {
          throw new Error(String(response.error));
        }
      }

      onSaved(trimmedName);
    } catch (saveError) {
      setError(
        saveError instanceof Error
          ? saveError.message
          : "Failed to save template.",
      );
    } finally {
      setIsSaving(false);
    }
  };

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit close button
    <div className="modal-overlay" onClick={onClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit close button */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="template-editor">
          <div className="template-editor__header">
            <div>
              <h3>{title}</h3>
              <p>
                Maintain selector rules directly here, then use the Visual
                Builder when you want live page analysis.
              </p>
            </div>
            <button
              type="button"
              className="btn btn--secondary btn--small"
              onClick={onClose}
              disabled={isSaving}
            >
              Close
            </button>
          </div>

          <div className="template-editor__body">
            <section className="template-editor__section">
              <label htmlFor="template-editor-name">Template name</label>
              <input
                id="template-editor-name"
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="my-template"
              />
            </section>

            <section className="template-editor__section">
              <div className="template-editor__section-header">
                <div>
                  <h4>Selector rules</h4>
                  <p>These drive the primary field extraction flow.</p>
                </div>
                <button
                  type="button"
                  className="btn btn--secondary btn--small"
                  onClick={() =>
                    setSelectors((current) => [
                      ...current,
                      createSelectorDraft(),
                    ])
                  }
                >
                  Add Selector
                </button>
              </div>

              <div className="template-editor__selectors">
                <div className="template-editor__selectors-header">
                  <span>Field</span>
                  <span>Selector</span>
                  <span>Attr</span>
                  <span>Options</span>
                </div>
                {selectors.map((selector) => (
                  <div
                    key={selector.id}
                    className="template-editor__selector-row"
                  >
                    <input
                      type="text"
                      value={selector.rule.name ?? ""}
                      onChange={(event) =>
                        updateSelectorField(
                          selector.id,
                          "name",
                          event.target.value,
                        )
                      }
                      placeholder="title"
                    />
                    <input
                      type="text"
                      value={selector.rule.selector ?? ""}
                      onChange={(event) =>
                        updateSelectorField(
                          selector.id,
                          "selector",
                          event.target.value,
                        )
                      }
                      placeholder="article h1"
                    />
                    <select
                      value={selector.rule.attr ?? "text"}
                      onChange={(event) =>
                        updateSelectorField(
                          selector.id,
                          "attr",
                          event.target.value,
                        )
                      }
                    >
                      <option value="text">text</option>
                      <option value="content">content</option>
                      <option value="href">href</option>
                      <option value="src">src</option>
                      <option value="alt">alt</option>
                      <option value="title">title</option>
                      <option value="value">value</option>
                    </select>
                    <div className="template-editor__selector-actions">
                      <label className="checkbox-label checkbox-label--small">
                        <input
                          type="checkbox"
                          checked={selector.rule.required ?? false}
                          onChange={(event) =>
                            updateSelectorField(
                              selector.id,
                              "required",
                              event.target.checked,
                            )
                          }
                        />
                        Required
                      </label>
                      <label className="checkbox-label checkbox-label--small">
                        <input
                          type="checkbox"
                          checked={selector.rule.all ?? false}
                          onChange={(event) =>
                            updateSelectorField(
                              selector.id,
                              "all",
                              event.target.checked,
                            )
                          }
                        />
                        All
                      </label>
                      <label className="checkbox-label checkbox-label--small">
                        <input
                          type="checkbox"
                          checked={selector.rule.trim ?? true}
                          onChange={(event) =>
                            updateSelectorField(
                              selector.id,
                              "trim",
                              event.target.checked,
                            )
                          }
                        />
                        Trim
                      </label>
                      <button
                        type="button"
                        className="btn btn--danger btn--small"
                        onClick={() =>
                          setSelectors((current) =>
                            current.filter(
                              (currentSelector) =>
                                currentSelector.id !== selector.id,
                            ),
                          )
                        }
                        disabled={selectors.length === 1}
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </section>

            <section className="template-editor__section">
              <div className="template-editor__section-header">
                <div>
                  <h4>Advanced rules</h4>
                  <p>
                    Optional JSON blocks for JSON-LD extraction, regex fallback,
                    and normalized field mapping.
                  </p>
                </div>
              </div>
              <div className="template-editor__json-grid">
                <label className="template-editor__json-field">
                  <span>JSON-LD rules</span>
                  <textarea
                    value={jsonldText}
                    onChange={(event) => setJsonldText(event.target.value)}
                    rows={8}
                    placeholder='[{"name":"author","type":"Article","path":"author.name"}]'
                  />
                </label>
                <label className="template-editor__json-field">
                  <span>Regex rules</span>
                  <textarea
                    value={regexText}
                    onChange={(event) => setRegexText(event.target.value)}
                    rows={8}
                    placeholder='[{"name":"price","pattern":"\\$([0-9.]+)","group":1,"source":"text"}]'
                  />
                </label>
                <label className="template-editor__json-field">
                  <span>Normalization settings</span>
                  <textarea
                    value={normalizeText}
                    onChange={(event) => setNormalizeText(event.target.value)}
                    rows={8}
                    placeholder='{"titleField":"title","descriptionField":"summary","metaFields":{"price":"product_price"}}'
                  />
                </label>
              </div>
            </section>

            {error && <div className="form-error">{error}</div>}
          </div>

          <div className="template-editor__footer">
            <button
              type="button"
              className="btn btn--secondary"
              onClick={onClose}
              disabled={isSaving}
            >
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={handleSave}
              disabled={isSaving}
            >
              {isSaving ? "Saving..." : saveLabel}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export function TemplateManager({
  templateNames,
  onTemplatesChanged,
  onOpenAIPreview,
  onOpenAIGenerator,
}: TemplateManagerProps) {
  const [selectedName, setSelectedName] = useState<string | null>(
    templateNames[0] ?? null,
  );
  const [selectedTemplate, setSelectedTemplate] =
    useState<TemplateDetail | null>(null);
  const [isLoadingDetail, setIsLoadingDetail] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [editorState, setEditorState] = useState<{
    mode: "create" | "edit" | "duplicate";
    originalName?: string;
    initialTemplate?: Template;
  } | null>(null);
  const [builderState, setBuilderState] = useState<{
    mode: "create" | "edit";
    initialTemplate?: Template;
  } | null>(null);
  const [isAIDebuggerOpen, setIsAIDebuggerOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    if (templateNames.length === 0) {
      setSelectedName(null);
      setSelectedTemplate(null);
      return;
    }

    if (!selectedName) {
      setSelectedName(templateNames[0] ?? null);
    }
  }, [selectedName, templateNames]);

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
          throw new Error(String(response.error));
        }

        if (!cancelled) {
          setSelectedTemplate(response.data ?? null);
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
  }, [selectedName]);

  const selectedTemplateData = selectedTemplate?.template;
  const isBuiltIn = selectedTemplate?.is_built_in ?? false;
  const hasTemplates = templateNames.length > 0;

  const stats = useMemo(() => {
    const builtInCount = templateNames.filter((name) =>
      BUILT_IN_TEMPLATE_NAMES.includes(
        name as (typeof BUILT_IN_TEMPLATE_NAMES)[number],
      ),
    ).length;

    return {
      total: templateNames.length,
      builtIn: builtInCount,
      custom: Math.max(templateNames.length - builtInCount, 0),
    };
  }, [templateNames]);

  const openEditor = (mode: "create" | "edit" | "duplicate") => {
    setEditorState({
      mode,
      originalName: selectedTemplateData?.name,
      initialTemplate:
        mode === "create"
          ? undefined
          : mode === "duplicate"
            ? {
                ...selectedTemplateData,
                name: selectedTemplateData?.name
                  ? getDuplicateName(selectedTemplateData.name)
                  : "",
              }
            : selectedTemplateData,
    });
  };

  const handleDelete = async () => {
    if (!selectedTemplateData?.name || isBuiltIn) {
      return;
    }

    if (
      !window.confirm(
        `Delete the "${selectedTemplateData.name}" template? This only removes the saved custom template.`,
      )
    ) {
      return;
    }

    setIsDeleting(true);
    setDetailError(null);

    try {
      const response = await deleteTemplate({
        baseUrl: getApiBaseUrl(),
        path: { name: selectedTemplateData.name },
      });
      if (response.error) {
        throw new Error(String(response.error));
      }

      onTemplatesChanged();
      setSelectedName((current) =>
        current === selectedTemplateData.name
          ? (templateNames.find((name) => name !== selectedTemplateData.name) ??
            null)
          : current,
      );
    } catch (error) {
      setDetailError(
        error instanceof Error ? error.message : "Failed to delete template.",
      );
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <>
      <section className="panel" style={{ marginTop: 16 }}>
        <div className="template-manager__hero">
          <div>
            <h3>Extraction Template Library</h3>
            <p>
              Use this page as the control plane for template inventory. Inspect
              saved templates, preview AI extraction against a real page, edit
              selector rules, duplicate built-ins into custom variants, and jump
              into the Visual Builder when you need live DOM analysis.
            </p>
          </div>
          <div className="template-manager__hero-actions">
            <button
              type="button"
              className="btn btn--secondary"
              onClick={() => openEditor("create")}
            >
              Create Template
            </button>
            <button
              type="button"
              className="btn btn--secondary"
              onClick={() => setBuilderState({ mode: "create" })}
            >
              Open Visual Builder
            </button>
            <button
              type="button"
              className="btn btn--secondary"
              onClick={onOpenAIPreview}
            >
              Preview Extraction with AI
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={onOpenAIGenerator}
            >
              Generate Template with AI
            </button>
          </div>
        </div>

        <div className="template-manager__stats">
          <div className="template-manager__stat">
            <span>Total templates</span>
            <strong>{stats.total}</strong>
          </div>
          <div className="template-manager__stat">
            <span>Custom templates</span>
            <strong>{stats.custom}</strong>
          </div>
          <div className="template-manager__stat">
            <span>Built-in templates</span>
            <strong>{stats.builtIn}</strong>
          </div>
        </div>
      </section>

      <section className="panel" style={{ marginTop: 16 }}>
        {hasTemplates ? (
          <div className="template-manager">
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
                      onClick={() => setSelectedName(name)}
                    >
                      <div className="template-manager__list-item-top">
                        <strong>{name}</strong>
                        <span
                          className={`badge ${templateKind === "Built-in" ? "running" : "success"}`}
                        >
                          {templateKind}
                        </span>
                      </div>
                      <span>Open details and management actions</span>
                    </button>
                  </li>
                );
              })}
            </ul>

            <div className="template-manager__detail">
              {isLoadingDetail ? (
                <div className="template-manager__empty">
                  Loading template details…
                </div>
              ) : detailError ? (
                <div className="template-manager__empty template-manager__empty--error">
                  {detailError}
                </div>
              ) : selectedTemplateData ? (
                <>
                  <div className="template-manager__detail-header">
                    <div>
                      <div className="template-manager__detail-eyebrow">
                        <span
                          className={`badge ${isBuiltIn ? "running" : "success"}`}
                        >
                          {isBuiltIn ? "Built-in template" : "Custom template"}
                        </span>
                      </div>
                      <h3>{selectedTemplateData.name}</h3>
                      <p>{describeTemplate(selectedTemplate)}</p>
                    </div>
                    <div className="template-manager__detail-actions">
                      <button
                        type="button"
                        className="btn btn--secondary btn--small"
                        onClick={() =>
                          setBuilderState({
                            mode: "edit",
                            initialTemplate: selectedTemplateData,
                          })
                        }
                      >
                        Edit in Visual Builder
                      </button>
                      <button
                        type="button"
                        className="btn btn--secondary btn--small"
                        onClick={() => setIsAIDebuggerOpen(true)}
                      >
                        Debug with AI
                      </button>
                      {isBuiltIn ? (
                        <button
                          type="button"
                          className="btn btn--secondary btn--small"
                          onClick={() => openEditor("duplicate")}
                        >
                          Duplicate
                        </button>
                      ) : (
                        <>
                          <button
                            type="button"
                            className="btn btn--secondary btn--small"
                            onClick={() => openEditor("edit")}
                          >
                            Edit
                          </button>
                          <button
                            type="button"
                            className="btn btn--secondary btn--small"
                            onClick={() => openEditor("duplicate")}
                          >
                            Duplicate
                          </button>
                          <button
                            type="button"
                            className="btn btn--danger btn--small"
                            onClick={handleDelete}
                            disabled={isDeleting}
                          >
                            {isDeleting ? "Deleting..." : "Delete"}
                          </button>
                        </>
                      )}
                    </div>
                  </div>

                  <div className="template-manager__detail-grid">
                    <section className="template-manager__card">
                      <div className="template-manager__card-header">
                        <h4>Selector rules</h4>
                        <span>
                          {selectedTemplateData.selectors?.length ?? 0}
                        </span>
                      </div>
                      {selectedTemplateData.selectors?.length ? (
                        <div className="template-manager__rule-list">
                          {selectedTemplateData.selectors.map((rule) => (
                            <div
                              key={ruleKey(rule)}
                              className="template-manager__rule"
                            >
                              <div className="template-manager__rule-top">
                                <strong>{rule.name ?? "Unnamed field"}</strong>
                                <span>{rule.attr ?? "text"}</span>
                              </div>
                              <code>{rule.selector}</code>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="template-manager__empty">
                          No selector rules configured.
                        </div>
                      )}
                    </section>

                    <section className="template-manager__card">
                      <div className="template-manager__card-header">
                        <h4>Advanced extraction</h4>
                      </div>
                      <dl className="template-manager__meta-list">
                        <div>
                          <dt>JSON-LD rules</dt>
                          <dd>{selectedTemplateData.jsonld?.length ?? 0}</dd>
                        </div>
                        <div>
                          <dt>Regex rules</dt>
                          <dd>{selectedTemplateData.regex?.length ?? 0}</dd>
                        </div>
                        <div>
                          <dt>Normalization</dt>
                          <dd>
                            {selectedTemplateData.normalize
                              ? "Configured"
                              : "Not configured"}
                          </dd>
                        </div>
                      </dl>
                      {selectedTemplateData.normalize && (
                        <pre className="template-manager__json-preview">
                          {formatJSON(selectedTemplateData.normalize)}
                        </pre>
                      )}
                    </section>
                  </div>
                </>
              ) : (
                <div className="template-manager__empty">
                  Select a template to inspect it.
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="template-manager__empty-state">
            <h3>No extraction templates saved yet</h3>
            <p>
              Start with AI preview against a real page, use the Visual Builder
              for live selector capture, create a manual template, or use AI
              generation to bootstrap from a real page.
            </p>
            <div className="template-manager__hero-actions">
              <button
                type="button"
                className="btn btn--secondary"
                onClick={() => openEditor("create")}
              >
                Create Template
              </button>
              <button
                type="button"
                className="btn btn--secondary"
                onClick={() => setBuilderState({ mode: "create" })}
              >
                Open Visual Builder
              </button>
              <button
                type="button"
                className="btn btn--secondary"
                onClick={onOpenAIPreview}
              >
                Preview Extraction with AI
              </button>
              <button
                type="button"
                className="btn btn--primary"
                onClick={onOpenAIGenerator}
              >
                Generate Template with AI
              </button>
            </div>
          </div>
        )}
      </section>

      {editorState && (
        <TemplateEditorModal
          mode={editorState.mode}
          originalName={editorState.originalName}
          initialTemplate={editorState.initialTemplate}
          onClose={() => setEditorState(null)}
          onSaved={(savedName) => {
            setEditorState(null);
            onTemplatesChanged();
            setSelectedName(savedName);
          }}
        />
      )}

      <AITemplateDebugger
        isOpen={isAIDebuggerOpen}
        template={selectedTemplateData ?? null}
        onClose={() => setIsAIDebuggerOpen(false)}
        onTemplateSaved={() => {
          setIsAIDebuggerOpen(false);
          onTemplatesChanged();
        }}
      />

      {builderState?.mode === "edit" && builderState.initialTemplate && (
        // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
        // biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit cancel button
        <div className="modal-overlay" onClick={() => setBuilderState(null)}>
          {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit cancel button */}
          <div
            className="modal-content"
            onClick={(event) => event.stopPropagation()}
          >
            <VisualSelectorBuilder
              initialTemplate={builderState.initialTemplate}
              onSave={(template) => {
                setBuilderState(null);
                setSelectedName(template.name ?? null);
                onTemplatesChanged();
              }}
              onCancel={() => setBuilderState(null)}
            />
          </div>
        </div>
      )}

      {builderState?.mode === "create" && (
        // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
        // biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit cancel button
        <div className="modal-overlay" onClick={() => setBuilderState(null)}>
          {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit cancel button */}
          <div
            className="modal-content"
            onClick={(event) => event.stopPropagation()}
          >
            <VisualSelectorBuilder
              onSave={(template) => {
                setBuilderState(null);
                setSelectedName(template.name ?? null);
                onTemplatesChanged();
              }}
              onCancel={() => setBuilderState(null)}
            />
          </div>
        </div>
      )}
    </>
  );
}
