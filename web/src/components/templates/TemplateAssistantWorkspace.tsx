/**
 * Purpose: Embed AI-powered template generation and debugging into the Templates workspace rail.
 * Responsibilities: Collect inline AI inputs, call the template AI endpoints, preserve optional image context, and let operators explicitly apply suggestions into the current draft workspace.
 * Scope: Templates-route AI workspace assistance only; persistence stays with the main workspace save flow.
 * Usage: Mounted by `TemplateManager` for the generate/debug rail tabs.
 * Invariants/Assumptions: AI results never auto-save, URL and HTML modes remain mutually exclusive, and image attachments are request-scoped only.
 */

import { useMemo, useState } from "react";

import {
  aiTemplateDebug,
  aiTemplateGenerate,
  type AiExtractTemplateDebugResponse,
  type AiExtractTemplateGenerateResponse,
  type ComponentStatus,
  type Template,
  type TemplateDetail,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import {
  buildAIAuthoringRequestContext,
  createAIAuthoringBrowserRuntimeState,
  updateAIAuthoringHeadlessState,
  updateAIAuthoringPlaywrightState,
  updateAIAuthoringVisualState,
} from "../../lib/ai-authoring-browser-runtime";
import type { AttachedAIImage } from "../../lib/ai-image-utils";
import { getApiErrorMessage } from "../../lib/api-errors";
import { describeAICapability, AIUnavailableNotice } from "../ai-assistant";
import { AIImageAttachments } from "../AIImageAttachments";
import { BrowserExecutionControls } from "../BrowserExecutionControls";

type AssistantMode = "generate" | "debug";
type SourceMode = "url" | "html";

interface TemplateAssistantWorkspaceProps {
  mode: AssistantMode;
  draftTemplate: Template;
  url: string;
  aiStatus?: ComponentStatus | null;
  onUrlChange: (value: string) => void;
  onApplyTemplate: (template: Template) => void;
}

function extractGeneratedTemplate(
  response: AiExtractTemplateGenerateResponse | null,
): Template | null {
  if (!response?.template) {
    return null;
  }

  const candidate = response.template as Template | TemplateDetail;
  if (candidate && typeof candidate === "object" && "template" in candidate) {
    return candidate.template ?? null;
  }

  return candidate as Template;
}

export function TemplateAssistantWorkspace({
  mode,
  draftTemplate,
  url,
  aiStatus = null,
  onUrlChange,
  onApplyTemplate,
}: TemplateAssistantWorkspaceProps) {
  const [source, setSource] = useState<SourceMode>("url");
  const [html, setHtml] = useState("");
  const [description, setDescription] = useState("");
  const [sampleFields, setSampleFields] = useState("");
  const [instructions, setInstructions] = useState("");
  const [images, setImages] = useState<AttachedAIImage[]>([]);
  const [headless, setHeadless] = useState(false);
  const [playwright, setPlaywright] = useState(false);
  const [visual, setVisual] = useState(false);
  const [isWorking, setIsWorking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [generated, setGenerated] =
    useState<AiExtractTemplateGenerateResponse | null>(null);
  const [debugged, setDebugged] =
    useState<AiExtractTemplateDebugResponse | null>(null);

  const generatedTemplate = useMemo(
    () => extractGeneratedTemplate(generated),
    [generated],
  );
  const aiCapability = describeAICapability(
    aiStatus,
    "Edit the template manually in the main workspace.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const interactionsDisabled = isWorking || aiUnavailable;

  const validate = () => {
    if (source === "url") {
      if (!url.trim()) {
        return "Target URL is required.";
      }
      try {
        new URL(url.trim());
      } catch {
        return "Please enter a valid URL.";
      }
    }

    if (source === "html" && !html.trim()) {
      return "HTML is required when using pasted HTML mode.";
    }

    if (mode === "generate" && !description.trim()) {
      return "Description is required.";
    }

    return null;
  };

  const resetResult = () => {
    setError(null);
    setGenerated(null);
    setDebugged(null);
  };

  const handleGenerate = async () => {
    if (aiUnavailable) {
      return;
    }
    const validationError = validate();
    if (validationError) {
      setError(validationError);
      return;
    }

    setIsWorking(true);
    setError(null);
    setGenerated(null);

    try {
      const response = await aiTemplateGenerate({
        baseUrl: getApiBaseUrl(),
        body: {
          ...buildAIAuthoringRequestContext({
            source,
            url,
            html,
            images,
            state: { headless, playwright, visual },
          }),
          description: description.trim(),
          sample_fields: sampleFields
            .split(",")
            .map((field) => field.trim())
            .filter(Boolean),
        },
      });

      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to generate template."),
        );
      }

      setGenerated(
        (response.data as AiExtractTemplateGenerateResponse) ?? null,
      );
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Failed to generate template.",
      );
    } finally {
      setIsWorking(false);
    }
  };

  const handleDebug = async () => {
    if (aiUnavailable) {
      return;
    }
    const validationError = validate();
    if (validationError) {
      setError(validationError);
      return;
    }

    setIsWorking(true);
    setError(null);
    setDebugged(null);

    try {
      const response = await aiTemplateDebug({
        baseUrl: getApiBaseUrl(),
        body: {
          ...buildAIAuthoringRequestContext({
            source,
            url,
            html,
            images,
            state: { headless, playwright, visual },
          }),
          template: draftTemplate,
          ...(instructions.trim() ? { instructions: instructions.trim() } : {}),
        },
      });

      if (response.error) {
        throw new Error(
          getApiErrorMessage(response.error, "Failed to debug template."),
        );
      }

      setDebugged((response.data as AiExtractTemplateDebugResponse) ?? null);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Failed to debug template.",
      );
    } finally {
      setIsWorking(false);
    }
  };

  const routeResult = mode === "generate" ? generated : debugged;

  return (
    <div className="template-assistant-panel">
      {aiUnavailableMessage ? (
        <div style={{ marginBottom: 16 }}>
          <AIUnavailableNotice message={aiUnavailableMessage} />
        </div>
      ) : null}

      <fieldset
        disabled={interactionsDisabled}
        style={{ border: 0, margin: 0, minInlineSize: 0, padding: 0 }}
      >
        <div className="template-assistant-panel__header">
          <div>
            <h4>{mode === "generate" ? "AI generator" : "AI debugger"}</h4>
            <p>
              {mode === "generate"
                ? "Generate a candidate template, review it in place, and explicitly apply it to the workspace."
                : "Debug the current draft, inspect the issues, and explicitly apply the suggested fix."}
            </p>
          </div>
        </div>

        {mode === "debug" ? (
          <div className="template-assistant-panel__callout">
            Debugging <code>{draftTemplate.name || "unsaved-workspace"}</code>
          </div>
        ) : null}

        <div className="form-group">
          <span className="form-label">Source</span>
          <div className="template-assistant-panel__source-toggle">
            <button
              type="button"
              className={`btn btn--secondary btn--small ${source === "url" ? "is-active" : ""}`}
              onClick={() => {
                setSource("url");
                resetResult();
              }}
              disabled={isWorking}
            >
              Fetch URL
            </button>
            <button
              type="button"
              className={`btn btn--secondary btn--small ${source === "html" ? "is-active" : ""}`}
              onClick={() => {
                setSource("html");
                const cleared = createAIAuthoringBrowserRuntimeState();
                setHeadless(cleared.headless);
                setPlaywright(cleared.playwright);
                setVisual(cleared.visual);
                resetResult();
              }}
              disabled={isWorking}
            >
              Paste HTML
            </button>
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="template-assistant-url" className="form-label">
            {source === "url" ? "Target URL" : "Page URL (optional)"}
          </label>
          <input
            id="template-assistant-url"
            type="url"
            value={url}
            onChange={(event) => {
              onUrlChange(event.target.value);
              resetResult();
            }}
            className="form-input"
            disabled={isWorking}
          />
        </div>

        {source === "html" ? (
          <div className="form-group">
            <label htmlFor="template-assistant-html" className="form-label">
              HTML
            </label>
            <textarea
              id="template-assistant-html"
              value={html}
              onChange={(event) => {
                setHtml(event.target.value);
                resetResult();
              }}
              rows={8}
              className="form-textarea font-mono text-xs"
              disabled={isWorking}
            />
          </div>
        ) : null}

        {mode === "generate" ? (
          <>
            <div className="form-group">
              <label
                htmlFor="template-assistant-description"
                className="form-label"
              >
                Description
              </label>
              <textarea
                id="template-assistant-description"
                value={description}
                onChange={(event) => {
                  setDescription(event.target.value);
                  resetResult();
                }}
                rows={3}
                className="form-textarea"
                disabled={isWorking}
              />
            </div>

            <div className="form-group">
              <label htmlFor="template-assistant-fields" className="form-label">
                Sample fields
              </label>
              <input
                id="template-assistant-fields"
                type="text"
                value={sampleFields}
                onChange={(event) => {
                  setSampleFields(event.target.value);
                  resetResult();
                }}
                className="form-input"
                disabled={isWorking}
              />
            </div>
          </>
        ) : (
          <div className="form-group">
            <label
              htmlFor="template-assistant-instructions"
              className="form-label"
            >
              Repair instructions
            </label>
            <textarea
              id="template-assistant-instructions"
              value={instructions}
              onChange={(event) => {
                setInstructions(event.target.value);
                resetResult();
              }}
              rows={3}
              className="form-textarea"
              disabled={isWorking}
            />
          </div>
        )}

        <AIImageAttachments
          images={images}
          onChange={(nextImages) => {
            setImages(nextImages);
            resetResult();
          }}
          disabled={interactionsDisabled}
          disabledReason={aiUnavailableMessage}
        />

        {source === "url" ? (
          <div className="template-assistant-panel__fetch-options space-y-3">
            <BrowserExecutionControls
              headless={headless}
              setHeadless={(value) => {
                const next = updateAIAuthoringHeadlessState(
                  { headless, playwright, visual },
                  value,
                );
                setHeadless(next.headless);
                setPlaywright(next.playwright);
                setVisual(next.visual);
                resetResult();
              }}
              usePlaywright={playwright}
              setUsePlaywright={(value) => {
                const next = updateAIAuthoringPlaywrightState(
                  { headless, playwright, visual },
                  value,
                );
                setHeadless(next.headless);
                setPlaywright(next.playwright);
                setVisual(next.visual);
                resetResult();
              }}
              headlessLabel="Use headless browser"
              playwrightLabel="Use Playwright"
              helperText="Enable headless to unlock Playwright for template authoring."
              showTimeout={false}
              disabled={isWorking}
            />

            <label className="checkbox-label checkbox-label--small">
              <input
                type="checkbox"
                checked={visual}
                onChange={(event) => {
                  const next = updateAIAuthoringVisualState(
                    { headless, playwright, visual },
                    event.target.checked,
                  );
                  setHeadless(next.headless);
                  setPlaywright(next.playwright);
                  setVisual(next.visual);
                  resetResult();
                }}
                disabled={isWorking}
              />
              Include screenshot context
            </label>
          </div>
        ) : null}

        {error ? <div className="form-error">{error}</div> : null}

        <div className="template-assistant-panel__actions">
          <button
            type="button"
            className="btn btn--primary"
            onClick={mode === "generate" ? handleGenerate : handleDebug}
            disabled={interactionsDisabled}
            title={aiUnavailableMessage ?? undefined}
          >
            {isWorking
              ? mode === "generate"
                ? "Generating..."
                : "Debugging..."
              : mode === "generate"
                ? "Generate Template"
                : "Debug Template"}
          </button>
        </div>

        {routeResult?.explanation ? (
          <div className="template-assistant-panel__callout">
            {routeResult.explanation}
          </div>
        ) : null}

        {routeResult?.route_id ||
        routeResult?.provider ||
        routeResult?.model ? (
          <div className="template-assistant-panel__route-info">
            <h5>AI Route</h5>
            <dl>
              {routeResult.route_id ? (
                <div>
                  <dt>Route</dt>
                  <dd>{routeResult.route_id}</dd>
                </div>
              ) : null}
              {routeResult.provider ? (
                <div>
                  <dt>Provider</dt>
                  <dd>{routeResult.provider}</dd>
                </div>
              ) : null}
              {routeResult.model ? (
                <div>
                  <dt>Model</dt>
                  <dd>{routeResult.model}</dd>
                </div>
              ) : null}
              <div>
                <dt>Visual context</dt>
                <dd>{routeResult.visual_context_used ? "Used" : "Not used"}</dd>
              </div>
            </dl>
          </div>
        ) : null}

        {mode === "generate" && generatedTemplate ? (
          <div className="template-assistant-panel__result">
            <div className="template-assistant-panel__template-preview">
              <strong>{generatedTemplate.name || "Generated template"}</strong>
              <ul>
                {(generatedTemplate.selectors ?? []).map((selector) => (
                  <li
                    key={`${selector.name ?? "field"}-${selector.selector ?? "selector"}`}
                  >
                    <span>{selector.name ?? "Unnamed field"}</span>
                    <code>{selector.selector}</code>
                  </li>
                ))}
              </ul>
            </div>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => onApplyTemplate(generatedTemplate)}
              disabled={aiUnavailable}
              title={aiUnavailableMessage ?? undefined}
            >
              Apply to workspace
            </button>
          </div>
        ) : null}

        {mode === "debug" && debugged ? (
          <div className="template-assistant-panel__result">
            {debugged.issues && debugged.issues.length > 0 ? (
              <div className="template-assistant-panel__issues">
                <h5>Detected issues</h5>
                <ul>
                  {debugged.issues.map((issue) => (
                    <li key={issue}>{issue}</li>
                  ))}
                </ul>
              </div>
            ) : null}

            {debugged.extracted_fields ? (
              <div className="template-assistant-panel__json-block">
                <h5>Current extracted fields</h5>
                <pre>{JSON.stringify(debugged.extracted_fields, null, 2)}</pre>
              </div>
            ) : null}

            {debugged.suggested_template ? (
              <>
                <div className="template-assistant-panel__json-block">
                  <h5>Suggested template</h5>
                  <pre>
                    {JSON.stringify(debugged.suggested_template, null, 2)}
                  </pre>
                </div>
                <button
                  type="button"
                  className="btn btn--primary"
                  onClick={() =>
                    onApplyTemplate(debugged.suggested_template as Template)
                  }
                  disabled={aiUnavailable}
                  title={aiUnavailableMessage ?? undefined}
                >
                  Apply suggestion
                </button>
              </>
            ) : null}
          </div>
        ) : null}
      </fieldset>
    </div>
  );
}

export default TemplateAssistantWorkspace;
