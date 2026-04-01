/**
 * Purpose: Render the reusable section blocks that make up the export schedule authoring dialog.
 * Responsibilities: Present the dialog shell, filter/configuration sections, transform and shape controls, retry fields, and footer actions while keeping labels and copy stable.
 * Scope: Export schedule form presentation only; AI assistant visibility state and modal orchestration stay in `ExportScheduleForm`.
 * Usage: Compose these sections from `ExportScheduleForm` to keep the main dialog component focused on state derivation and assistant wiring.
 * Invariants/Assumptions: Existing field ids, labels, and explanatory copy remain stable for tests and operator workflows, and the form stays fully controlled by parent-owned `formData` plus `onChange`.
 */

import type { ReactNode } from "react";

import {
  clearShapeFormData,
  clearTransformFormData,
} from "../../lib/export-schedule-utils";
import type { ExportScheduleFormProps } from "../../types/export-schedule";
import { AIUnavailableNotice } from "../ai-assistant";
import { PromotionDraftNotice } from "../promotion/PromotionDraftNotice";

const JOB_KIND_OPTIONS = [
  { value: "scrape", label: "Scrape" },
  { value: "crawl", label: "Crawl" },
  { value: "research", label: "Research" },
] as const;

const JOB_STATUS_OPTIONS = [
  { value: "completed", label: "Completed" },
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
  { value: "canceled", label: "Canceled" },
] as const;

const FORMAT_OPTIONS = [
  { value: "json", label: "JSON" },
  { value: "jsonl", label: "JSON Lines" },
  { value: "md", label: "Markdown" },
  { value: "csv", label: "CSV" },
  { value: "xlsx", label: "Excel (XLSX)" },
] as const;

const DESTINATION_OPTIONS = [
  { value: "local", label: "Local File" },
  { value: "webhook", label: "Webhook" },
] as const;

const sectionStyle = {
  marginBottom: 24,
  padding: 16,
  backgroundColor: "var(--bg-alt)",
  borderRadius: 8,
} as const;

const sectionTitleStyle = { margin: "0 0 12px 0", fontSize: 14 } as const;
const fieldLabelStyle = {
  display: "block",
  marginBottom: 4,
  fontSize: 13,
} as const;
const mutedInlineStyle = { color: "var(--muted)", fontSize: 13 } as const;
const codeTextareaStyle = {
  width: "100%",
  fontFamily: "monospace",
  fontSize: 12,
} as const;

export function ExportScheduleDialogShell({
  formError,
  aiAssistantMessage,
  promotionSeed,
  onClearPromotionSeed,
  onOpenSourceJob,
  isEditing,
  onCancel,
  onSubmit,
  children,
}: Pick<
  ExportScheduleFormProps,
  | "formError"
  | "promotionSeed"
  | "onClearPromotionSeed"
  | "onOpenSourceJob"
  | "isEditing"
  | "onCancel"
  | "onSubmit"
> & {
  aiAssistantMessage: string | null;
  children: ReactNode;
}) {
  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        backgroundColor: "rgba(0, 0, 0, 0.7)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        padding: 20,
      }}
    >
      <div
        className="panel"
        style={{
          maxWidth: 760,
          width: "100%",
          maxHeight: "90vh",
          overflow: "auto",
        }}
      >
        <div
          className="row"
          style={{
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 16,
          }}
        >
          <h3 style={{ margin: 0 }}>
            {isEditing ? "Edit Export Schedule" : "Create Export Schedule"}
          </h3>
          <button type="button" onClick={onCancel} className="secondary">
            Cancel
          </button>
        </div>

        {formError ? (
          <div
            style={{
              padding: 12,
              backgroundColor: "rgba(239, 68, 68, 0.1)",
              borderRadius: 8,
              color: "#ef4444",
              marginBottom: 16,
            }}
          >
            {formError}
          </div>
        ) : null}

        {promotionSeed ? (
          <div style={{ marginBottom: 16 }}>
            <PromotionDraftNotice
              title="Recurring export draft seeded from a verified job"
              description="This draft automates export for future matching jobs, not a rerun cadence for the source job."
              seed={promotionSeed}
              onOpenSourceJob={onOpenSourceJob}
              onClear={onClearPromotionSeed}
            />
          </div>
        ) : null}

        <form
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit();
          }}
        >
          {aiAssistantMessage ? (
            <div style={{ marginBottom: 16 }}>
              <AIUnavailableNotice message={aiAssistantMessage} />
            </div>
          ) : null}

          {children}
        </form>
      </div>
    </div>
  );
}

export function ExportScheduleBasicInfoSection({
  formData,
  onChange,
}: Pick<ExportScheduleFormProps, "formData" | "onChange">) {
  return (
    <div style={{ marginBottom: 24 }}>
      <h4 style={sectionTitleStyle}>Basic Info</h4>
      <div style={{ marginBottom: 16 }}>
        <label
          htmlFor="schedule-name"
          style={{ display: "block", marginBottom: 4 }}
        >
          Name <span style={{ color: "#ef4444" }}>*</span>
        </label>
        <input
          id="schedule-name"
          type="text"
          value={formData.name}
          onChange={(event) => onChange({ name: event.target.value })}
          placeholder="My Export Schedule"
          required
          style={{ width: "100%" }}
        />
      </div>

      <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <input
          type="checkbox"
          checked={formData.enabled}
          onChange={(event) => onChange({ enabled: event.target.checked })}
        />
        Enabled
      </label>
    </div>
  );
}

export function ExportScheduleFiltersSection({
  formData,
  onChange,
  toggleJobKind,
  toggleJobStatus,
}: Pick<ExportScheduleFormProps, "formData" | "onChange"> & {
  toggleJobKind: (kind: string) => void;
  toggleJobStatus: (status: string) => void;
}) {
  return (
    <div style={sectionStyle}>
      <h4 style={sectionTitleStyle}>Job Filters</h4>

      <div style={{ marginBottom: 16 }}>
        <div style={{ display: "block", marginBottom: 8, fontSize: 13 }}>
          Job Kinds (empty = all kinds)
        </div>
        <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
          {JOB_KIND_OPTIONS.map((kind) => (
            <label
              key={kind.value}
              style={{ display: "flex", alignItems: "center", gap: 6 }}
            >
              <input
                type="checkbox"
                checked={formData.filterJobKinds.includes(
                  kind.value as (typeof formData.filterJobKinds)[number],
                )}
                onChange={() => toggleJobKind(kind.value)}
              />
              {kind.label}
            </label>
          ))}
        </div>
      </div>

      <div style={{ marginBottom: 16 }}>
        <div style={{ display: "block", marginBottom: 8, fontSize: 13 }}>
          Job Status (empty = completed only)
        </div>
        <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
          {JOB_STATUS_OPTIONS.map((status) => (
            <label
              key={status.value}
              style={{ display: "flex", alignItems: "center", gap: 6 }}
            >
              <input
                type="checkbox"
                checked={formData.filterJobStatus.includes(
                  status.value as (typeof formData.filterJobStatus)[number],
                )}
                onChange={() => toggleJobStatus(status.value)}
              />
              {status.label}
            </label>
          ))}
        </div>
      </div>

      <div style={{ marginBottom: 16 }}>
        <label htmlFor="filter-tags" style={fieldLabelStyle}>
          Tags (one per line, all must match)
        </label>
        <textarea
          id="filter-tags"
          value={formData.filterTags}
          onChange={(event) => onChange({ filterTags: event.target.value })}
          placeholder="production&#10;critical"
          rows={2}
          style={codeTextareaStyle}
        />
      </div>

      <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <input
          type="checkbox"
          checked={formData.filterHasResults}
          onChange={(event) =>
            onChange({ filterHasResults: event.target.checked })
          }
        />
        Only match jobs with non-empty results
      </label>
    </div>
  );
}

export function ExportScheduleConfigSection({
  formData,
  onChange,
}: Pick<ExportScheduleFormProps, "formData" | "onChange">) {
  return (
    <div style={sectionStyle}>
      <h4 style={sectionTitleStyle}>Export Configuration</h4>

      <div className="row" style={{ gap: 16, marginBottom: 16 }}>
        <div style={{ flex: 1 }}>
          <label
            htmlFor="export-format"
            style={{ display: "block", marginBottom: 4 }}
          >
            Format
          </label>
          <select
            id="export-format"
            value={formData.format}
            onChange={(event) =>
              onChange({
                format: event.target.value as typeof formData.format,
              })
            }
            style={{ width: "100%" }}
          >
            {FORMAT_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        <div style={{ flex: 1 }}>
          <label
            htmlFor="destination-type"
            style={{ display: "block", marginBottom: 4 }}
          >
            Destination Type
          </label>
          <select
            id="destination-type"
            value={formData.destinationType}
            onChange={(event) =>
              onChange({
                destinationType: event.target
                  .value as typeof formData.destinationType,
              })
            }
            style={{ width: "100%" }}
          >
            {DESTINATION_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div style={{ marginBottom: 16 }}>
        <label
          htmlFor="path-template"
          style={{ display: "block", marginBottom: 4 }}
        >
          Path Template
        </label>
        <input
          id="path-template"
          type="text"
          value={formData.pathTemplate}
          onChange={(event) => onChange({ pathTemplate: event.target.value })}
          placeholder="exports/{kind}/{job_id}.{format}"
          style={{ width: "100%" }}
        />
        <small style={{ color: "var(--muted)" }}>
          Variables: {"{"}job_id{"}"}, {"{"}timestamp{"}"}, {"{"}kind{"}"},{" "}
          {"{"}format{"}"}
        </small>
      </div>

      {formData.destinationType === "local" ? (
        <div
          style={{
            marginTop: 16,
            padding: 16,
            backgroundColor: "var(--bg)",
            borderRadius: 6,
          }}
        >
          <h5 style={{ margin: "0 0 12px 0", fontSize: 13 }}>
            Local File Configuration
          </h5>
          <div>
            <label htmlFor="local-path" style={fieldLabelStyle}>
              Path Template <span style={{ color: "#ef4444" }}>*</span>
            </label>
            <input
              id="local-path"
              type="text"
              value={formData.localPath}
              onChange={(event) => onChange({ localPath: event.target.value })}
              placeholder="exports/{kind}/{job_id}.{format}"
              style={{ width: "100%" }}
            />
            <small style={{ color: "var(--muted)" }}>
              Saved schedules can only write within DATA_DIR/exports.
            </small>
          </div>
        </div>
      ) : null}

      {formData.destinationType === "webhook" ? (
        <div
          style={{
            marginTop: 16,
            padding: 16,
            backgroundColor: "var(--bg)",
            borderRadius: 6,
          }}
        >
          <h5 style={{ margin: "0 0 12px 0", fontSize: 13 }}>
            Webhook Configuration
          </h5>
          <div>
            <label htmlFor="webhook-url" style={fieldLabelStyle}>
              Webhook URL <span style={{ color: "#ef4444" }}>*</span>
            </label>
            <input
              id="webhook-url"
              type="url"
              value={formData.webhookUrl}
              onChange={(event) => onChange({ webhookUrl: event.target.value })}
              placeholder="https://api.example.com/webhook"
              style={{ width: "100%" }}
            />
          </div>
        </div>
      ) : null}
    </div>
  );
}

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
