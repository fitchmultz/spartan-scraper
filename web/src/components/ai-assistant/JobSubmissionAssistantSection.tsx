/**
 * Purpose: Embed route-aware AI extraction help directly into the new-job workflow.
 * Responsibilities: Mirror the current job form context into the assistant shell, run AI extraction previews, and provide explicit apply actions back into the shared form controller.
 * Scope: `/jobs/new` assistant behavior only.
 * Usage: Mount from `JobSubmissionContainer` alongside the guided wizard and expert forms.
 * Invariants/Assumptions: AI preview never auto-mutates the job form, URL and HTML preview modes stay mutually exclusive, and the job form remains the canonical owner of submission state.
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import {
  aiExtractPreview,
  type AiExtractPreviewResponse,
  type FieldValue,
} from "../../api";
import type { FormController } from "../../hooks/useFormState";
import {
  buildPresetConfig,
  type JobDraftLocalState,
} from "../../lib/job-drafts";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import {
  toAIImagePayloads,
  type AttachedAIImage,
} from "../../lib/ai-image-utils";
import type { JobType } from "../../types/presets";
import { AIImageAttachments } from "../AIImageAttachments";
import type { AssistantContext } from "./AIAssistantProvider";
import { AIAssistantPanel } from "./AIAssistantPanel";
import { useAIAssistant } from "./useAIAssistant";

type PreviewSource = "url" | "html";
type PreviewMode = "natural_language" | "schema_guided";

interface JobSubmissionAssistantSectionProps {
  activeTab: JobType;
  form: FormController;
  localState: JobDraftLocalState;
}

interface PreviewState {
  source: PreviewSource;
  url: string;
  html: string;
  mode: PreviewMode;
  prompt: string;
  schemaText: string;
  fields: string;
  images: AttachedAIImage[];
  headless: boolean;
  playwright: boolean;
  visual: boolean;
  isLoading: boolean;
  result: AiExtractPreviewResponse | null;
  error: string | null;
}

const SCHEMA_GUIDED_PLACEHOLDER = `{
  "title": "Example item",
  "price": "$19.99"
}`;

function formatFieldValues(field: FieldValue) {
  return field.values && field.values.length > 0 ? field.values : [];
}

function buildContext(
  activeTab: JobType,
  form: FormController,
  localState: JobDraftLocalState,
): Extract<AssistantContext, { surface: "job-submission" }> {
  const snapshot = buildPresetConfig(activeTab, form, localState);

  return {
    surface: "job-submission",
    jobType: activeTab,
    url:
      activeTab === "scrape"
        ? localState.scrape.url || undefined
        : activeTab === "crawl"
          ? localState.crawl.url || undefined
          : undefined,
    query:
      activeTab === "research"
        ? localState.research.query || undefined
        : undefined,
    templateName: form.extractTemplate || undefined,
    formSnapshot: snapshot as Record<string, unknown>,
  };
}

function createInitialState(
  context: Extract<AssistantContext, { surface: "job-submission" }>,
  form: FormController,
): PreviewState {
  return {
    source: "url",
    url: context.url ?? "",
    html: "",
    mode: form.aiExtractMode,
    prompt: form.aiExtractPrompt,
    schemaText: form.aiExtractSchema || SCHEMA_GUIDED_PLACEHOLDER,
    fields: form.aiExtractFields,
    images: [],
    headless: form.headless,
    playwright: form.usePlaywright,
    visual: false,
    isLoading: false,
    result: null,
    error: null,
  };
}

function isInitialStateBlank(state: PreviewState): boolean {
  return (
    !state.url &&
    !state.html &&
    !state.prompt &&
    !state.fields &&
    state.images.length === 0 &&
    !state.result &&
    !state.error
  );
}

export function JobSubmissionAssistantSection({
  activeTab,
  form,
  localState,
}: JobSubmissionAssistantSectionProps) {
  const { setContext } = useAIAssistant();

  const assistantContext = useMemo(
    () => buildContext(activeTab, form, localState),
    [activeTab, form, localState],
  );

  const [state, setState] = useState<PreviewState>(() =>
    createInitialState(assistantContext, form),
  );
  const previousJobTypeRef = useRef(activeTab);

  useEffect(() => {
    setContext(assistantContext);
  }, [assistantContext, setContext]);

  useEffect(() => {
    setState((previous) => {
      if (previousJobTypeRef.current !== activeTab) {
        previousJobTypeRef.current = activeTab;
        return createInitialState(assistantContext, form);
      }

      if (!isInitialStateBlank(previous)) {
        return previous;
      }

      return createInitialState(assistantContext, form);
    });
  }, [activeTab, assistantContext, form]);

  const resultFields = Object.entries(state.result?.fields ?? {});

  const loadCurrentFormSettings = () => {
    setState(createInitialState(assistantContext, form));
  };

  const validate = () => {
    if (state.source === "url") {
      if (!state.url.trim()) {
        return "URL is required.";
      }
      try {
        new URL(state.url.trim());
      } catch {
        return "Please enter a valid URL.";
      }
    }

    if (state.source === "html" && !state.html.trim()) {
      return "HTML is required when using pasted HTML mode.";
    }

    if (state.mode === "natural_language" && !state.prompt.trim()) {
      return "Extraction instructions are required.";
    }

    if (state.mode === "schema_guided") {
      if (!state.schemaText.trim()) {
        return "Schema example JSON is required.";
      }
      try {
        const parsed = JSON.parse(state.schemaText);
        if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") {
          return "Schema example must be a JSON object.";
        }
      } catch {
        return "Schema example must be valid JSON.";
      }
    }

    return null;
  };

  const handlePreview = async () => {
    const validationError = validate();
    if (validationError) {
      setState((previous) => ({ ...previous, error: validationError }));
      return;
    }

    setState((previous) => ({
      ...previous,
      isLoading: true,
      error: null,
      result: null,
    }));

    try {
      const fields = state.fields
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean);

      const response = await aiExtractPreview({
        baseUrl: getApiBaseUrl(),
        body: {
          ...(state.source === "url" ? { url: state.url.trim() } : {}),
          ...(state.source === "html" ? { html: state.html } : {}),
          ...(state.source === "html" && state.url.trim()
            ? { url: state.url.trim() }
            : {}),
          mode: state.mode,
          ...(state.mode === "natural_language"
            ? { prompt: state.prompt.trim() }
            : {
                schema: JSON.parse(state.schemaText) as Record<string, unknown>,
              }),
          ...(fields.length > 0 ? { fields } : {}),
          ...(state.images.length > 0
            ? { images: toAIImagePayloads(state.images) }
            : {}),
          ...(state.source === "url"
            ? {
                headless: state.headless,
                playwright: state.headless ? state.playwright : false,
                visual: state.visual,
              }
            : {}),
        },
      });

      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to generate preview."),
        );
      }

      setState((previous) => ({
        ...previous,
        isLoading: false,
        result: (response.data as AiExtractPreviewResponse) ?? null,
      }));
    } catch (error) {
      setState((previous) => ({
        ...previous,
        isLoading: false,
        error:
          error instanceof Error ? error.message : "Failed to run AI preview.",
      }));
    }
  };

  const applyToForm = () => {
    form.setAIExtractEnabled(true);
    form.setAIExtractMode(state.mode);
    form.setAIExtractFields(state.fields);
    if (state.mode === "natural_language") {
      form.setAIExtractPrompt(state.prompt.trim());
    } else {
      form.setAIExtractSchema(state.schemaText);
    }
  };

  const applyReturnedFieldNames = () => {
    if (!state.result?.fields) {
      return;
    }

    const fieldNames = Object.keys(state.result.fields);
    const nextValue = fieldNames.join(", ");
    form.setAIExtractFields(nextValue);
    setState((previous) => ({
      ...previous,
      fields: nextValue,
    }));
  };

  return (
    <JobSubmissionAssistantShell
      assistantContext={assistantContext}
      state={state}
      resultFields={resultFields}
      onChangeState={setState}
      onLoadCurrentFormSettings={loadCurrentFormSettings}
      onRunPreview={handlePreview}
      onApplyToForm={applyToForm}
      onApplyReturnedFieldNames={applyReturnedFieldNames}
    />
  );
}

interface JobSubmissionAssistantShellProps {
  assistantContext: Extract<AssistantContext, { surface: "job-submission" }>;
  state: PreviewState;
  resultFields: [string, FieldValue][];
  onChangeState: Dispatch<SetStateAction<PreviewState>>;
  onLoadCurrentFormSettings: () => void;
  onRunPreview: () => Promise<void>;
  onApplyToForm: () => void;
  onApplyReturnedFieldNames: () => void;
}

function JobSubmissionAssistantShell({
  assistantContext,
  state,
  resultFields,
  onChangeState,
  onLoadCurrentFormSettings,
  onRunPreview,
  onApplyToForm,
  onApplyReturnedFieldNames,
}: JobSubmissionAssistantShellProps) {
  return (
    <AIAssistantPanel
      title="Job submission assistant"
      routeLabel="/jobs/new"
      suggestedActions={
        <>
          <button
            type="button"
            className="secondary"
            onClick={onLoadCurrentFormSettings}
          >
            Load current form settings
          </button>
          <button type="button" onClick={() => void onRunPreview()}>
            {state.isLoading ? "Running preview…" : "Run preview"}
          </button>
          <button type="button" className="secondary" onClick={onApplyToForm}>
            Apply settings to form
          </button>
          {state.result ? (
            <button
              type="button"
              className="secondary"
              onClick={onApplyReturnedFieldNames}
            >
              Use returned field names
            </button>
          ) : null}
        </>
      }
    >
      <div className="form-group">
        <span className="form-label">Source</span>
        <div className="template-assistant-panel__source-toggle">
          <button
            type="button"
            className={`btn btn--secondary btn--small ${state.source === "url" ? "is-active" : ""}`}
            onClick={() =>
              onChangeState((previous) => ({
                ...previous,
                source: "url",
                error: null,
              }))
            }
            disabled={state.isLoading}
          >
            Fetch URL
          </button>
          <button
            type="button"
            className={`btn btn--secondary btn--small ${state.source === "html" ? "is-active" : ""}`}
            onClick={() =>
              onChangeState((previous) => ({
                ...previous,
                source: "html",
                error: null,
                headless: false,
                playwright: false,
                visual: false,
              }))
            }
            disabled={state.isLoading}
          >
            Paste HTML
          </button>
        </div>
        {assistantContext.query ? (
          <p className="form-help">
            Research query in progress:{" "}
            <strong>{assistantContext.query}</strong>
          </p>
        ) : null}
      </div>

      <div className="form-group">
        <label htmlFor="job-assistant-url" className="form-label">
          {state.source === "url" ? "Target URL" : "Page URL (optional)"}
        </label>
        <input
          id="job-assistant-url"
          type="url"
          className="form-input"
          value={state.url}
          onChange={(event) =>
            onChangeState((previous) => ({
              ...previous,
              url: event.target.value,
            }))
          }
          disabled={state.isLoading}
        />
      </div>

      {state.source === "html" ? (
        <div className="form-group">
          <label htmlFor="job-assistant-html" className="form-label">
            HTML
          </label>
          <textarea
            id="job-assistant-html"
            rows={8}
            className="form-textarea font-mono text-xs"
            value={state.html}
            onChange={(event) =>
              onChangeState((previous) => ({
                ...previous,
                html: event.target.value,
              }))
            }
            disabled={state.isLoading}
          />
        </div>
      ) : null}

      <div className="form-group">
        <span className="form-label">Extraction mode</span>
        <div className="template-assistant-panel__source-toggle">
          <button
            type="button"
            className={`btn btn--secondary btn--small ${state.mode === "natural_language" ? "is-active" : ""}`}
            onClick={() =>
              onChangeState((previous) => ({
                ...previous,
                mode: "natural_language",
                error: null,
              }))
            }
          >
            Natural language
          </button>
          <button
            type="button"
            className={`btn btn--secondary btn--small ${state.mode === "schema_guided" ? "is-active" : ""}`}
            onClick={() =>
              onChangeState((previous) => ({
                ...previous,
                mode: "schema_guided",
                error: null,
              }))
            }
          >
            Schema guided
          </button>
        </div>
      </div>

      {state.mode === "natural_language" ? (
        <div className="form-group">
          <label htmlFor="job-assistant-prompt" className="form-label">
            Extraction instructions
          </label>
          <textarea
            id="job-assistant-prompt"
            rows={4}
            className="form-textarea"
            value={state.prompt}
            onChange={(event) =>
              onChangeState((previous) => ({
                ...previous,
                prompt: event.target.value,
              }))
            }
            disabled={state.isLoading}
          />
        </div>
      ) : (
        <div className="form-group">
          <label htmlFor="job-assistant-schema" className="form-label">
            Schema example JSON
          </label>
          <textarea
            id="job-assistant-schema"
            rows={8}
            className="form-textarea font-mono text-sm"
            value={state.schemaText}
            onChange={(event) =>
              onChangeState((previous) => ({
                ...previous,
                schemaText: event.target.value,
              }))
            }
            disabled={state.isLoading}
          />
        </div>
      )}

      <div className="form-group">
        <label htmlFor="job-assistant-fields" className="form-label">
          Specific fields
        </label>
        <input
          id="job-assistant-fields"
          type="text"
          className="form-input"
          value={state.fields}
          onChange={(event) =>
            onChangeState((previous) => ({
              ...previous,
              fields: event.target.value,
            }))
          }
          disabled={state.isLoading}
        />
      </div>

      <AIImageAttachments
        images={state.images}
        onChange={(images) =>
          onChangeState((previous) => ({
            ...previous,
            images,
          }))
        }
        disabled={state.isLoading}
      />

      {state.source === "url" ? (
        <div className="template-assistant-panel__fetch-options">
          <label className="checkbox-label checkbox-label--small">
            <input
              type="checkbox"
              checked={state.headless}
              onChange={(event) =>
                onChangeState((previous) => ({
                  ...previous,
                  headless: event.target.checked,
                  playwright: event.target.checked
                    ? previous.playwright
                    : false,
                  visual: event.target.checked ? previous.visual : false,
                }))
              }
              disabled={state.isLoading}
            />
            Use headless browser
          </label>

          <label className="checkbox-label checkbox-label--small">
            <input
              type="checkbox"
              checked={state.playwright}
              onChange={(event) =>
                onChangeState((previous) => ({
                  ...previous,
                  playwright: event.target.checked,
                }))
              }
              disabled={!state.headless || state.isLoading}
            />
            Use Playwright
          </label>

          <label className="checkbox-label checkbox-label--small">
            <input
              type="checkbox"
              checked={state.visual}
              onChange={(event) =>
                onChangeState((previous) => ({
                  ...previous,
                  visual: event.target.checked,
                  headless: event.target.checked ? true : previous.headless,
                }))
              }
              disabled={state.isLoading}
            />
            Include screenshot context
          </label>
        </div>
      ) : null}

      {state.error ? <div className="form-error">{state.error}</div> : null}

      {state.result ? (
        <div className="template-assistant-panel__result">
          <div className="results-viewer__badge-row">
            <span>
              Confidence {Math.round((state.result.confidence ?? 0) * 100)}%
            </span>
            <span>
              Tokens {(state.result.tokens_used ?? 0).toLocaleString()}
            </span>
            <span>Cache {state.result.cached ? "Hit" : "Miss"}</span>
          </div>

          {state.result.explanation ? (
            <div className="template-assistant-panel__callout">
              {state.result.explanation}
            </div>
          ) : null}

          <div className="template-assistant-panel__json-block">
            <h5>Extracted fields</h5>
            <div className="job-list">
              {resultFields.map(([name, field]) => (
                <div key={name} className="job-item">
                  <strong>{name}</strong>
                  {formatFieldValues(field).map((value) => (
                    <div key={`${name}-${value}`}>{value}</div>
                  ))}
                  {field.rawObject ? <pre>{field.rawObject}</pre> : null}
                </div>
              ))}
            </div>
          </div>
        </div>
      ) : null}
    </AIAssistantPanel>
  );
}
