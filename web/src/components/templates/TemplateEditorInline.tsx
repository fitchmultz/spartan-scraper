/**
 * Purpose: Render the inline template editor for the Templates workspace.
 * Responsibilities: Expose editable template metadata, selector rows, advanced JSON inputs, and save/reset controls without modal overlays.
 * Scope: Inline authoring UI only; persistence, template loading, and AI workflows stay in parent workspace components.
 * Usage: Mounted by `TemplateManager` inside the center editor pane.
 * Invariants/Assumptions: The parent owns draft state, built-in templates are read-only until duplicated, and selector row controls must stay keyboard accessible.
 */

import type { SelectorRule } from "../../api";
import { getTemplateDraftValidationIssues } from "./useTemplateMutationActions";
import type { TemplateDraftState } from "./templateRouteControllerShared";

interface TemplateEditorInlineProps {
  draft: TemplateDraftState;
  readOnly: boolean;
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  notice: string | null;
  onNameChange: (value: string) => void;
  onUpdateSelector: (
    selectorId: string,
    key: keyof SelectorRule,
    value: string | boolean,
  ) => void;
  onAddSelector: () => void;
  onRemoveSelector: (selectorId: string) => void;
  onJsonldTextChange: (value: string) => void;
  onRegexTextChange: (value: string) => void;
  onNormalizeTextChange: (value: string) => void;
  onSave: () => void;
  onReset: () => void;
  onClose?: () => void;
  onDiscard?: () => void;
  closeLabel?: string;
  discardLabel?: string;
}

export function TemplateEditorInline({
  draft,
  readOnly,
  isDirty,
  isSaving,
  error,
  notice,
  onNameChange,
  onUpdateSelector,
  onAddSelector,
  onRemoveSelector,
  onJsonldTextChange,
  onRegexTextChange,
  onNormalizeTextChange,
  onSave,
  onReset,
  onClose,
  onDiscard,
  closeLabel = "Close",
  discardLabel = "Discard draft",
}: TemplateEditorInlineProps) {
  const saveDisabledIssues = readOnly
    ? []
    : getTemplateDraftValidationIssues(draft);
  const saveDisabledReason = saveDisabledIssues[0] ?? null;

  return (
    <div className="template-editor-inline">
      <section className="template-editor-inline__section">
        <div className="template-editor-inline__section-header">
          <div>
            <h4>Template metadata</h4>
            <p>
              Keep the name stable for saved workflows and set up the extraction
              rules that the preview rail will exercise.
            </p>
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="template-editor-name" className="form-label">
            Template name
          </label>
          <input
            id="template-editor-name"
            type="text"
            value={draft.name}
            onChange={(event) => onNameChange(event.target.value)}
            placeholder="my-template"
            className="form-input"
            readOnly={readOnly}
            disabled={readOnly || isSaving}
          />
        </div>
      </section>

      <section className="template-editor-inline__section">
        <div className="template-editor-inline__section-header">
          <div>
            <h4>Selector rules</h4>
            <p>
              Define field names, selectors, and extraction options without
              leaving the workspace.
            </p>
          </div>
          {!readOnly ? (
            <button
              type="button"
              className="btn btn--secondary btn--small"
              onClick={onAddSelector}
              disabled={isSaving}
            >
              Add Selector
            </button>
          ) : null}
        </div>

        <div className="template-editor-inline__selectors">
          <div className="template-editor-inline__selectors-header">
            <span>Field</span>
            <span>Selector</span>
            <span>Attr</span>
            <span>Options</span>
          </div>

          {draft.selectors.map((selector) => (
            <div
              key={selector.id}
              className="template-editor-inline__selector-row"
            >
              <div className="form-group">
                <label
                  className="sr-only"
                  htmlFor={`selector-name-${selector.id}`}
                >
                  Field name
                </label>
                <input
                  id={`selector-name-${selector.id}`}
                  type="text"
                  value={selector.rule.name ?? ""}
                  onChange={(event) =>
                    onUpdateSelector(selector.id, "name", event.target.value)
                  }
                  placeholder="title"
                  className="form-input"
                  readOnly={readOnly}
                  disabled={readOnly || isSaving}
                />
              </div>

              <div className="form-group">
                <label
                  className="sr-only"
                  htmlFor={`selector-query-${selector.id}`}
                >
                  CSS selector
                </label>
                <input
                  id={`selector-query-${selector.id}`}
                  type="text"
                  value={selector.rule.selector ?? ""}
                  onChange={(event) =>
                    onUpdateSelector(
                      selector.id,
                      "selector",
                      event.target.value,
                    )
                  }
                  placeholder="article h1"
                  className="form-input"
                  readOnly={readOnly}
                  disabled={readOnly || isSaving}
                />
              </div>

              <div className="form-group">
                <label
                  className="sr-only"
                  htmlFor={`selector-attr-${selector.id}`}
                >
                  Attribute
                </label>
                <select
                  id={`selector-attr-${selector.id}`}
                  value={selector.rule.attr ?? "text"}
                  onChange={(event) =>
                    onUpdateSelector(selector.id, "attr", event.target.value)
                  }
                  className="form-input"
                  disabled={readOnly || isSaving}
                >
                  <option value="text">text</option>
                  <option value="content">content</option>
                  <option value="href">href</option>
                  <option value="src">src</option>
                  <option value="alt">alt</option>
                  <option value="title">title</option>
                  <option value="value">value</option>
                </select>
              </div>

              <div className="template-editor-inline__selector-actions">
                <div className="form-group template-editor-inline__selector-meta">
                  <label
                    className="sr-only"
                    htmlFor={`selector-join-${selector.id}`}
                  >
                    Join string
                  </label>
                  <input
                    id={`selector-join-${selector.id}`}
                    type="text"
                    value={selector.rule.join ?? ""}
                    onChange={(event) =>
                      onUpdateSelector(selector.id, "join", event.target.value)
                    }
                    placeholder="Join values (optional)"
                    className="form-input"
                    readOnly={readOnly}
                    disabled={readOnly || isSaving}
                  />
                </div>

                <div className="template-editor-inline__selector-toggles">
                  <label className="checkbox-label checkbox-label--small">
                    <input
                      type="checkbox"
                      checked={selector.rule.required ?? false}
                      onChange={(event) =>
                        onUpdateSelector(
                          selector.id,
                          "required",
                          event.target.checked,
                        )
                      }
                      disabled={readOnly || isSaving}
                    />
                    Required
                  </label>
                  <label className="checkbox-label checkbox-label--small">
                    <input
                      type="checkbox"
                      checked={selector.rule.all ?? false}
                      onChange={(event) =>
                        onUpdateSelector(
                          selector.id,
                          "all",
                          event.target.checked,
                        )
                      }
                      disabled={readOnly || isSaving}
                    />
                    All
                  </label>
                  <label className="checkbox-label checkbox-label--small">
                    <input
                      type="checkbox"
                      checked={selector.rule.trim ?? true}
                      onChange={(event) =>
                        onUpdateSelector(
                          selector.id,
                          "trim",
                          event.target.checked,
                        )
                      }
                      disabled={readOnly || isSaving}
                    />
                    Trim
                  </label>
                </div>

                {!readOnly ? (
                  <button
                    type="button"
                    className="btn btn--danger btn--small"
                    onClick={() => onRemoveSelector(selector.id)}
                    disabled={isSaving || draft.selectors.length === 1}
                  >
                    Remove
                  </button>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="template-editor-inline__section">
        <div className="template-editor-inline__section-header">
          <div>
            <h4>Advanced extraction</h4>
            <p>
              Add optional JSON-LD, regex, or normalization rules when the
              selector layer needs structured fallbacks.
            </p>
          </div>
        </div>

        <div className="template-editor-inline__json-grid">
          <label className="template-editor-inline__json-field">
            <span>JSON-LD rules</span>
            <textarea
              value={draft.jsonldText}
              onChange={(event) => onJsonldTextChange(event.target.value)}
              rows={8}
              placeholder='[{"name":"author","type":"Article","path":"author.name"}]'
              className="form-textarea font-mono text-xs"
              readOnly={readOnly}
              disabled={readOnly || isSaving}
            />
          </label>

          <label className="template-editor-inline__json-field">
            <span>Regex rules</span>
            <textarea
              value={draft.regexText}
              onChange={(event) => onRegexTextChange(event.target.value)}
              rows={8}
              placeholder='[{"name":"price","pattern":"\\$([0-9.]+)","group":1,"source":"text"}]'
              className="form-textarea font-mono text-xs"
              readOnly={readOnly}
              disabled={readOnly || isSaving}
            />
          </label>

          <label className="template-editor-inline__json-field">
            <span>Normalization settings</span>
            <textarea
              value={draft.normalizeText}
              onChange={(event) => onNormalizeTextChange(event.target.value)}
              rows={8}
              placeholder='{"titleField":"title","descriptionField":"summary","metaFields":{"price":"product_price"}}'
              className="form-textarea font-mono text-xs"
              readOnly={readOnly}
              disabled={readOnly || isSaving}
            />
          </label>
        </div>
      </section>

      {(error || notice || readOnly || saveDisabledReason) && (
        <div className="template-editor-inline__status" aria-live="polite">
          {readOnly ? (
            <span>
              Built-in templates stay read-only in place. Duplicate this
              template to make changes.
            </span>
          ) : null}
          {!readOnly && saveDisabledIssues.length > 0 ? (
            <div className="template-editor-inline__status-blockers">
              <strong>Save stays disabled until you finish:</strong>
              <ul>
                {saveDisabledIssues.map((issue) => (
                  <li key={issue}>{issue}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {notice ? <span>{notice}</span> : null}
          {error ? <span className="form-error">{error}</span> : null}
        </div>
      )}

      <div className="template-editor-inline__footer">
        <span className="template-editor-inline__footer-copy">
          {readOnly
            ? "Preview or debug this template, then duplicate it into the workspace when you are ready to edit."
            : saveDisabledReason
              ? "Complete the save blockers above, then preview or save the reusable draft."
              : isDirty
                ? "Unsaved changes stay in the workspace until you explicitly save them."
                : "Your workspace is up to date."}
        </span>
        <div className="template-editor-inline__footer-actions">
          {!readOnly ? (
            <button
              type="button"
              className="btn btn--secondary"
              onClick={onReset}
              disabled={isSaving}
            >
              Reset Draft
            </button>
          ) : null}
          {!readOnly && onClose ? (
            <button
              type="button"
              className="btn btn--secondary"
              onClick={onClose}
              disabled={isSaving}
            >
              {closeLabel}
            </button>
          ) : null}
          {!readOnly && onDiscard ? (
            <button
              type="button"
              className="btn btn--secondary"
              onClick={onDiscard}
              disabled={isSaving}
            >
              {discardLabel}
            </button>
          ) : null}
          {!readOnly ? (
            <button
              type="button"
              className="btn btn--primary"
              onClick={onSave}
              disabled={isSaving || !!saveDisabledReason}
              title={saveDisabledReason ?? undefined}
            >
              {isSaving ? "Saving..." : "Save Template"}
            </button>
          ) : null}
        </div>
      </div>
    </div>
  );
}

export default TemplateEditorInline;
