/**
 * Purpose: Provide selector preview feedback alongside the inline template workspace.
 * Responsibilities: Collect a target URL, run selector tests for the current draft, and summarize sample matches without leaving the Templates route.
 * Scope: Preview-only workspace rail for template authoring.
 * Usage: Mounted by `TemplateManager` when the preview rail tab is active.
 * Invariants/Assumptions: Preview runs are explicit, selector testing uses the canonical API, and incomplete selector rows should be ignored instead of crashing the workspace.
 */

import { useMemo, useState } from "react";

import {
  testSelector,
  type SelectorRule,
  type Template,
  type TestSelectorResponse,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { ruleKey } from "./templateEditorUtils";

interface PreviewResult {
  rule: SelectorRule;
  result: TestSelectorResponse;
}

interface TemplatePreviewPaneProps {
  template: Template;
  url: string;
  onUrlChange: (value: string) => void;
}

export function TemplatePreviewPane({
  template,
  url,
  onUrlChange,
}: TemplatePreviewPaneProps) {
  const [headless, setHeadless] = useState(false);
  const [playwright, setPlaywright] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [results, setResults] = useState<PreviewResult[]>([]);

  const selectors = useMemo(
    () =>
      (template.selectors ?? []).filter(
        (rule) =>
          (rule.name?.trim().length ?? 0) > 0 &&
          (rule.selector?.trim().length ?? 0) > 0,
      ),
    [template.selectors],
  );

  const handleRunPreview = async () => {
    if (!url.trim()) {
      setError("Preview target URL is required.");
      return;
    }

    try {
      new URL(url.trim());
    } catch {
      setError("Please enter a valid preview URL.");
      return;
    }

    if (selectors.length === 0) {
      setError(
        "Add at least one complete selector rule before running preview.",
      );
      return;
    }

    setIsRunning(true);
    setError(null);

    try {
      const previewResults = await Promise.all(
        selectors.map(async (rule) => {
          const response = await testSelector({
            baseUrl: getApiBaseUrl(),
            body: {
              url: url.trim(),
              selector: rule.selector ?? "",
              ...(headless ? { headless: true } : {}),
              ...(playwright ? { playwright: true } : {}),
            },
          });

          if (response.error) {
            throw new Error(
              getApiErrorMessage(
                response.error,
                "Failed to run selector preview.",
              ),
            );
          }

          return {
            rule,
            result: response.data ?? {},
          } satisfies PreviewResult;
        }),
      );

      setResults(previewResults);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Failed to run selector preview.",
      );
      setResults([]);
    } finally {
      setIsRunning(false);
    }
  };

  return (
    <div className="template-preview-pane">
      <div className="template-preview-pane__header">
        <div>
          <h4>Preview</h4>
          <p>
            Run every complete selector against a real page and inspect the
            first matches before saving.
          </p>
        </div>
      </div>

      <div className="form-group">
        <label htmlFor="template-preview-url" className="form-label">
          Preview target URL
        </label>
        <input
          id="template-preview-url"
          type="url"
          value={url}
          onChange={(event) => onUrlChange(event.target.value)}
          placeholder="https://example.com/article"
          className="form-input"
          disabled={isRunning}
        />
      </div>

      <div className="template-preview-pane__result-top">
        <div className="template-preview-pane__fetch-options">
          <label className="checkbox-label checkbox-label--small">
            <input
              type="checkbox"
              checked={headless}
              onChange={(event) => {
                setHeadless(event.target.checked);
                if (!event.target.checked) {
                  setPlaywright(false);
                }
              }}
              disabled={isRunning}
            />
            Use headless browser
          </label>
          <label className="checkbox-label checkbox-label--small">
            <input
              type="checkbox"
              checked={playwright}
              onChange={(event) => setPlaywright(event.target.checked)}
              disabled={!headless || isRunning}
            />
            Use Playwright
          </label>
        </div>

        <button
          type="button"
          className="btn btn--primary"
          onClick={handleRunPreview}
          disabled={isRunning}
        >
          {isRunning ? "Running..." : "Run Preview"}
        </button>
      </div>

      {error ? <div className="form-error">{error}</div> : null}

      <div className="template-preview-pane__results">
        {results.length === 0 ? (
          <div className="template-preview-pane__empty">
            {selectors.length === 0
              ? "Complete at least one selector row to enable live preview."
              : "Run preview to inspect current selector matches."}
          </div>
        ) : (
          results.map(({ rule, result }) => (
            <article
              key={ruleKey(rule)}
              className="template-preview-pane__result"
            >
              <div className="template-preview-pane__result-top">
                <div>
                  <strong>{rule.name ?? "Unnamed field"}</strong>
                  <div className="template-preview-pane__result-meta">
                    <code>{rule.selector}</code>
                  </div>
                </div>
                <span className="badge success">
                  {result.matches ?? 0} match{result.matches === 1 ? "" : "es"}
                </span>
              </div>

              {result.error ? (
                <div className="form-error">{result.error}</div>
              ) : (result.elements?.length ?? 0) === 0 ? (
                <div className="template-preview-pane__empty">
                  No elements matched this selector.
                </div>
              ) : (
                <div className="template-preview-pane__samples">
                  {(result.elements ?? []).slice(0, 3).map((element) => (
                    <div
                      key={`${element.path ?? element.tag ?? "node"}-${element.text ?? ""}`}
                      className="template-preview-pane__sample"
                    >
                      <span className="template-preview-pane__sample-tag">
                        {element.tag ?? "node"}
                      </span>
                      <span>{element.text ?? "(no text)"}</span>
                    </div>
                  ))}
                </div>
              )}
            </article>
          ))
        )}
      </div>
    </div>
  );
}

export default TemplatePreviewPane;
