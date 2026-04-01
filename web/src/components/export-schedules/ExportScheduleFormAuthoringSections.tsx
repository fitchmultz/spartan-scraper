/**
 * Purpose: Render the transform, shape, retry, and footer action sections for export schedule authoring.
 * Responsibilities: Present advanced export authoring controls while keeping transform/shape guardrails and footer actions consistent.
 * Scope: Export schedule form presentation only; assistant visibility state and save orchestration stay in `ExportScheduleForm`.
 * Usage: Compose these sections from `ExportScheduleForm` through the local section barrel.
 * Invariants/Assumptions: Transform and shape authoring remain mutually exclusive, supported-format rules stay explicit, and submit/cancel actions remain stable.
 */

import {
  clearShapeFormData,
  clearTransformFormData,
} from "../../lib/export-schedule-utils";
import type { ExportScheduleFormProps } from "../../types/export-schedule";
import {
  codeTextareaStyle,
  fieldLabelStyle,
  mutedInlineStyle,
  sectionStyle,
  sectionTitleStyle,
} from "./exportScheduleFormShared";

export function ExportScheduleTransformSection({
  formData,
  onChange,
  transformSummary,
  stagedTransform,
  transformLockedByShape,
  aiAssistantUnavailable,
  aiAssistantMessage,
  onOpenTransformAssistant,
}: Pick<ExportScheduleFormProps, "formData" | "onChange"> & {
  transformSummary: string;
  stagedTransform: boolean;
  transformLockedByShape: boolean;
  aiAssistantUnavailable: boolean;
  aiAssistantMessage: string | null;
  onOpenTransformAssistant: () => void;
}) {
  return (
    <div style={sectionStyle}>
      <div
        className="row"
        style={{
          justifyContent: "space-between",
          alignItems: "center",
          gap: 12,
          marginBottom: 12,
        }}
      >
        <div>
          <h4 style={{ margin: 0, fontSize: 14 }}>Result Transform</h4>
          <div style={{ ...mutedInlineStyle, marginTop: 4 }}>
            Optionally project or reshape saved results with JMESPath or JSONata
            before recurring export runs.
          </div>
        </div>
        <div className="row" style={{ gap: 8, flexWrap: "wrap" }}>
          <div className="badge running">{transformSummary}</div>
          <button
            type="button"
            className="secondary"
            onClick={() => onChange(clearTransformFormData())}
            disabled={!stagedTransform}
          >
            Clear Transform
          </button>
          <button
            type="button"
            onClick={onOpenTransformAssistant}
            disabled={transformLockedByShape || aiAssistantUnavailable}
            title={aiAssistantMessage ?? undefined}
          >
            AI Suggest Transform
          </button>
        </div>
      </div>

      {transformLockedByShape ? (
        <div
          style={{
            padding: 12,
            borderRadius: 8,
            backgroundColor: "rgba(148, 163, 184, 0.12)",
            border: "1px solid rgba(148, 163, 184, 0.25)",
            color: "var(--muted)",
          }}
        >
          Export shaping is active for this schedule. Clear the shape before
          configuring a saved transform so recurring exports keep one
          deterministic projection contract.
        </div>
      ) : (
        <>
          <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
            <div style={{ flex: "0 0 220px" }}>
              <label htmlFor="transform-language" style={fieldLabelStyle}>
                Transform language
              </label>
              <select
                id="transform-language"
                value={formData.transformLanguage}
                onChange={(event) =>
                  onChange({
                    transformLanguage: event.target
                      .value as typeof formData.transformLanguage,
                  })
                }
                style={{ width: "100%" }}
              >
                <option value="jmespath">JMESPath</option>
                <option value="jsonata">JSONata</option>
              </select>
            </div>
          </div>

          <div style={{ marginTop: 16 }}>
            <label htmlFor="transform-expression" style={fieldLabelStyle}>
              Transform expression
            </label>
            <textarea
              id="transform-expression"
              value={formData.transformExpression}
              onChange={(event) =>
                onChange({ transformExpression: event.target.value })
              }
              rows={5}
              placeholder={
                formData.transformLanguage === "jmespath"
                  ? "{title: title, url: url, price: normalized.fields.price.values[0]}"
                  : '$.{"title": title, "url": url}'
              }
              style={codeTextareaStyle}
            />
            <small style={{ color: "var(--muted)" }}>
              Leave empty to export the canonical saved results without an
              additional transform.
            </small>
          </div>
        </>
      )}
    </div>
  );
}

export function ExportScheduleShapeSection({
  formData,
  onChange,
  shapeSummary,
  stagedShape,
  shapeSupported,
  shapeLockedByTransform,
  aiAssistantUnavailable,
  aiAssistantMessage,
  onOpenShapeAssistant,
}: Pick<ExportScheduleFormProps, "formData" | "onChange"> & {
  shapeSummary: string;
  stagedShape: boolean;
  shapeSupported: boolean;
  shapeLockedByTransform: boolean;
  aiAssistantUnavailable: boolean;
  aiAssistantMessage: string | null;
  onOpenShapeAssistant: () => void;
}) {
  return (
    <div style={sectionStyle}>
      <div
        className="row"
        style={{
          justifyContent: "space-between",
          alignItems: "center",
          gap: 12,
          marginBottom: 12,
        }}
      >
        <div>
          <h4 style={{ margin: 0, fontSize: 14 }}>Export Shaping</h4>
          <div style={{ ...mutedInlineStyle, marginTop: 4 }}>
            Customize recurring Markdown, CSV, and XLSX exports with a bounded
            field-selection workflow.
          </div>
        </div>
        <div className="row" style={{ gap: 8, flexWrap: "wrap" }}>
          <div className="badge running">{shapeSummary}</div>
          <button
            type="button"
            className="secondary"
            onClick={() => onChange(clearShapeFormData())}
            disabled={!stagedShape}
          >
            Clear Shape
          </button>
          <button
            type="button"
            onClick={onOpenShapeAssistant}
            disabled={
              !shapeSupported ||
              shapeLockedByTransform ||
              aiAssistantUnavailable
            }
            title={aiAssistantMessage ?? undefined}
          >
            AI Suggest Shape
          </button>
        </div>
      </div>

      {shapeLockedByTransform ? (
        <div
          style={{
            padding: 12,
            borderRadius: 8,
            backgroundColor: "rgba(148, 163, 184, 0.12)",
            border: "1px solid rgba(148, 163, 184, 0.25)",
            color: "var(--muted)",
          }}
        >
          This schedule already has a saved transform. Clear the transform
          before configuring export shaping so recurring exports keep one
          deterministic projection path.
        </div>
      ) : shapeSupported ? (
        <>
          <div
            style={{
              marginBottom: 16,
              padding: 12,
              borderRadius: 8,
              backgroundColor: "rgba(99, 102, 241, 0.08)",
              border: "1px solid rgba(99, 102, 241, 0.25)",
            }}
          >
            Use newline-separated field references. Add label overrides as
            <code style={{ marginLeft: 4 }}>field.key=Label</code>.
          </div>

          <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
            <div style={{ flex: "1 1 260px" }}>
              <label htmlFor="shape-top-level-fields" style={fieldLabelStyle}>
                Top-level fields
              </label>
              <textarea
                id="shape-top-level-fields"
                value={formData.shapeTopLevelFields}
                onChange={(event) =>
                  onChange({ shapeTopLevelFields: event.target.value })
                }
                rows={5}
                placeholder="url&#10;title&#10;status"
                style={codeTextareaStyle}
              />
            </div>
            <div style={{ flex: "1 1 260px" }}>
              <label htmlFor="shape-summary-fields" style={fieldLabelStyle}>
                Summary fields
              </label>
              <textarea
                id="shape-summary-fields"
                value={formData.shapeSummaryFields}
                onChange={(event) =>
                  onChange({ shapeSummaryFields: event.target.value })
                }
                rows={5}
                placeholder="title&#10;field.price"
                style={codeTextareaStyle}
              />
            </div>
          </div>

          <div
            className="row"
            style={{ gap: 16, flexWrap: "wrap", marginTop: 16 }}
          >
            <div style={{ flex: "1 1 260px" }}>
              <label htmlFor="shape-normalized-fields" style={fieldLabelStyle}>
                Normalized fields
              </label>
              <textarea
                id="shape-normalized-fields"
                value={formData.shapeNormalizedFields}
                onChange={(event) =>
                  onChange({ shapeNormalizedFields: event.target.value })
                }
                rows={5}
                placeholder="field.price&#10;field.plan"
                style={codeTextareaStyle}
              />
            </div>
            <div style={{ flex: "1 1 260px" }}>
              <label htmlFor="shape-evidence-fields" style={fieldLabelStyle}>
                Evidence fields
              </label>
              <textarea
                id="shape-evidence-fields"
                value={formData.shapeEvidenceFields}
                onChange={(event) =>
                  onChange({ shapeEvidenceFields: event.target.value })
                }
                rows={5}
                placeholder="evidence.url&#10;evidence.title"
                style={codeTextareaStyle}
              />
            </div>
          </div>

          <div style={{ marginTop: 16 }}>
            <label htmlFor="shape-field-labels" style={fieldLabelStyle}>
              Field labels
            </label>
            <textarea
              id="shape-field-labels"
              value={formData.shapeFieldLabels}
              onChange={(event) =>
                onChange({ shapeFieldLabels: event.target.value })
              }
              rows={4}
              placeholder="field.price=Price&#10;title=Page Title"
              style={codeTextareaStyle}
            />
          </div>

          <div
            className="row"
            style={{ gap: 16, flexWrap: "wrap", marginTop: 16 }}
          >
            <div style={{ flex: "1 1 180px" }}>
              <label htmlFor="shape-empty-value" style={fieldLabelStyle}>
                Empty value placeholder
              </label>
              <input
                id="shape-empty-value"
                type="text"
                value={formData.shapeEmptyValue}
                onChange={(event) =>
                  onChange({ shapeEmptyValue: event.target.value })
                }
                placeholder="—"
                style={{ width: "100%" }}
              />
            </div>
            <div style={{ flex: "1 1 180px" }}>
              <label htmlFor="shape-multi-value-join" style={fieldLabelStyle}>
                Multi-value join
              </label>
              <input
                id="shape-multi-value-join"
                type="text"
                value={formData.shapeMultiValueJoin}
                onChange={(event) =>
                  onChange({ shapeMultiValueJoin: event.target.value })
                }
                placeholder="; "
                style={{ width: "100%" }}
              />
            </div>
            <div style={{ flex: "1 1 220px" }}>
              <label htmlFor="shape-markdown-title" style={fieldLabelStyle}>
                Markdown title override
              </label>
              <input
                id="shape-markdown-title"
                type="text"
                value={formData.shapeMarkdownTitle}
                onChange={(event) =>
                  onChange({ shapeMarkdownTitle: event.target.value })
                }
                placeholder="Pricing Export"
                style={{ width: "100%" }}
              />
            </div>
          </div>
        </>
      ) : (
        <div
          style={{
            padding: 12,
            borderRadius: 8,
            backgroundColor: "rgba(148, 163, 184, 0.12)",
            border: "1px solid rgba(148, 163, 184, 0.25)",
            color: "var(--muted)",
          }}
        >
          JSON and JSON Lines exports always ship the full structured payload.
          Switch to Markdown, CSV, or XLSX to configure or apply bounded export
          shaping.
          {stagedShape ? (
            <div style={{ marginTop: 8, color: "var(--text)" }}>
              Shape fields are currently staged in the form but will not be
              saved unless you switch back to a supported format.
            </div>
          ) : null}
        </div>
      )}
    </div>
  );
}

export function ExportScheduleRetrySection({
  formData,
  onChange,
}: Pick<ExportScheduleFormProps, "formData" | "onChange">) {
  return (
    <div style={sectionStyle}>
      <h4 style={sectionTitleStyle}>Retry Configuration</h4>
      <div className="row" style={{ gap: 16 }}>
        <div style={{ flex: 1 }}>
          <label htmlFor="max-retries" style={fieldLabelStyle}>
            Max Retries
          </label>
          <input
            id="max-retries"
            type="number"
            min={0}
            step={1}
            value={formData.maxRetries}
            onChange={(event) =>
              onChange({
                maxRetries: Number.parseInt(event.target.value, 10) || 0,
              })
            }
            style={{ width: "100%" }}
          />
        </div>
        <div style={{ flex: 1 }}>
          <label htmlFor="base-delay" style={fieldLabelStyle}>
            Base Delay (ms)
          </label>
          <input
            id="base-delay"
            type="number"
            min={0}
            step={100}
            value={formData.baseDelayMs}
            onChange={(event) =>
              onChange({
                baseDelayMs: Number.parseInt(event.target.value, 10) || 0,
              })
            }
            style={{ width: "100%" }}
          />
        </div>
      </div>
    </div>
  );
}

export function ExportScheduleFormActions({
  formSubmitting,
  isEditing,
  onCancel,
}: Pick<ExportScheduleFormProps, "formSubmitting" | "isEditing" | "onCancel">) {
  return (
    <div
      className="row"
      style={{
        gap: 8,
        justifyContent: "flex-end",
        position: "sticky",
        bottom: 0,
        zIndex: 10,
        paddingTop: 16,
        marginTop: 24,
        borderTop: "1px solid var(--stroke)",
        background: "var(--panel)",
      }}
    >
      <button
        type="button"
        onClick={onCancel}
        className="secondary"
        disabled={formSubmitting}
      >
        Cancel
      </button>
      <button type="submit" disabled={formSubmitting}>
        {formSubmitting
          ? "Saving..."
          : isEditing
            ? "Update Schedule"
            : "Create Schedule"}
      </button>
    </div>
  );
}
