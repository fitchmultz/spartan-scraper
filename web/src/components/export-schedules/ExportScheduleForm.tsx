/**
 * ExportScheduleForm Component
 *
 * Renders the create/edit form for export schedules in a modal dialog.
 * Includes all form fields: name, enabled, filters, export config,
 * destination-specific settings, export shaping, and retry configuration.
 *
 * This component does NOT handle:
 * - API calls for saving export schedules (parent handles via onSubmit)
 * - Form state management (controlled via props)
 * - Modal visibility state
 *
 * @module components/export-schedules/ExportScheduleForm
 */

import { useEffect, useState } from "react";
import type { ExportScheduleFormProps } from "../../types/export-schedule";
import { AIExportShapeAssistant } from "../AIExportShapeAssistant";
import { AIExportTransformAssistant } from "../AIExportTransformAssistant";
import { AIUnavailableNotice, describeAICapability } from "../ai-assistant";
import {
  clearShapeFormData,
  clearTransformFormData,
  formDataToShapeConfig,
  formDataToTransformConfig,
  formatExportShapeSummary,
  formatExportTransformSummary,
  hasShapeFormData,
  hasTransformFormData,
  shapeConfigToFormData,
  supportsExportShapeFormat,
  transformConfigToFormData,
} from "../../lib/export-schedule-utils";

const JOB_KIND_OPTIONS = [
  { value: "scrape", label: "Scrape" },
  { value: "crawl", label: "Crawl" },
  { value: "research", label: "Research" },
];

const JOB_STATUS_OPTIONS = [
  { value: "completed", label: "Completed" },
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
  { value: "canceled", label: "Canceled" },
];

const FORMAT_OPTIONS = [
  { value: "json", label: "JSON" },
  { value: "jsonl", label: "JSON Lines" },
  { value: "md", label: "Markdown" },
  { value: "csv", label: "CSV" },
  { value: "xlsx", label: "Excel (XLSX)" },
];

const DESTINATION_OPTIONS = [
  { value: "local", label: "Local File" },
  { value: "webhook", label: "Webhook" },
];

/**
 * Form component for creating or editing an export schedule
 */
export function ExportScheduleForm({
  formData,
  formError,
  formSubmitting,
  isEditing,
  onChange,
  onSubmit,
  onCancel,
  aiStatus = null,
}: ExportScheduleFormProps) {
  const [showShapeAssistant, setShowShapeAssistant] = useState(false);
  const [showTransformAssistant, setShowTransformAssistant] = useState(false);
  const aiCapability = describeAICapability(
    aiStatus,
    "Configure transforms and shapes manually in this form.",
  );
  const aiAssistantUnavailable = aiCapability.unavailable;
  const aiAssistantMessage = aiCapability.message;
  const shapeSupported = supportsExportShapeFormat(formData.format);
  const stagedShape = hasShapeFormData(formData);
  const stagedTransform = hasTransformFormData(formData);
  const currentTransform = formDataToTransformConfig(formData);
  const transformSummary = formatExportTransformSummary(currentTransform);
  const transformActive = stagedTransform;
  const shapeLockedByTransform = transformActive;
  const transformLockedByShape = stagedShape;
  const currentShape =
    shapeSupported && !shapeLockedByTransform
      ? formDataToShapeConfig(formData)
      : undefined;
  const shapeSummary = shapeSupported
    ? shapeLockedByTransform
      ? "Disabled by transform"
      : formatExportShapeSummary(currentShape)
    : stagedShape
      ? "Staged (unsupported format)"
      : "Default";

  useEffect(() => {
    if (!aiAssistantUnavailable) {
      return;
    }
    setShowShapeAssistant(false);
    setShowTransformAssistant(false);
  }, [aiAssistantUnavailable]);

  const toggleJobKind = (kind: string) => {
    const current = formData.filterJobKinds;
    if (current.includes(kind as (typeof current)[number])) {
      onChange({
        filterJobKinds: current.filter((k) => k !== kind),
      });
    } else {
      onChange({
        filterJobKinds: [...current, kind as (typeof current)[number]],
      });
    }
  };

  const toggleJobStatus = (status: string) => {
    const current = formData.filterJobStatus;
    if (current.includes(status as (typeof current)[number])) {
      onChange({
        filterJobStatus: current.filter((s) => s !== status),
      });
    } else {
      onChange({
        filterJobStatus: [...current, status as (typeof current)[number]],
      });
    }
  };

  return (
    <>
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

          {formError && (
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
          )}

          <form
            onSubmit={(e) => {
              e.preventDefault();
              onSubmit();
            }}
          >
            {aiAssistantMessage ? (
              <div style={{ marginBottom: 16 }}>
                <AIUnavailableNotice message={aiAssistantMessage} />
              </div>
            ) : null}

            {/* Basic Info */}
            <div style={{ marginBottom: 24 }}>
              <h4 style={{ margin: "0 0 12px 0", fontSize: 14 }}>Basic Info</h4>
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
                  onChange={(e) => onChange({ name: e.target.value })}
                  placeholder="My Export Schedule"
                  required
                  style={{ width: "100%" }}
                />
              </div>

              <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <input
                  type="checkbox"
                  checked={formData.enabled}
                  onChange={(e) => onChange({ enabled: e.target.checked })}
                />
                Enabled
              </label>
            </div>

            {/* Filters */}
            <div
              style={{
                marginBottom: 24,
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <h4 style={{ margin: "0 0 12px 0", fontSize: 14 }}>
                Job Filters
              </h4>

              <div style={{ marginBottom: 16 }}>
                <div
                  style={{ display: "block", marginBottom: 8, fontSize: 13 }}
                >
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
                <div
                  style={{ display: "block", marginBottom: 8, fontSize: 13 }}
                >
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
                <label
                  htmlFor="filter-tags"
                  style={{ display: "block", marginBottom: 4, fontSize: 13 }}
                >
                  Tags (one per line, all must match)
                </label>
                <textarea
                  id="filter-tags"
                  value={formData.filterTags}
                  onChange={(e) => onChange({ filterTags: e.target.value })}
                  placeholder="production&#10;critical"
                  rows={2}
                  style={{
                    width: "100%",
                    fontFamily: "monospace",
                    fontSize: 12,
                  }}
                />
              </div>

              <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <input
                  type="checkbox"
                  checked={formData.filterHasResults}
                  onChange={(e) =>
                    onChange({ filterHasResults: e.target.checked })
                  }
                />
                Only match jobs with non-empty results
              </label>
            </div>

            {/* Export Config */}
            <div
              style={{
                marginBottom: 24,
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <h4 style={{ margin: "0 0 12px 0", fontSize: 14 }}>
                Export Configuration
              </h4>

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
                    onChange={(e) =>
                      onChange({
                        format: e.target.value as typeof formData.format,
                      })
                    }
                    style={{ width: "100%" }}
                  >
                    {FORMAT_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
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
                    onChange={(e) =>
                      onChange({
                        destinationType: e.target
                          .value as typeof formData.destinationType,
                      })
                    }
                    style={{ width: "100%" }}
                  >
                    {DESTINATION_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
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
                  onChange={(e) => onChange({ pathTemplate: e.target.value })}
                  placeholder="exports/{kind}/{job_id}.{format}"
                  style={{ width: "100%" }}
                />
                <small style={{ color: "var(--muted)" }}>
                  Variables: {"{"}job_id{"}"}, {"{"}timestamp{"}"}, {"{"}kind
                  {"}"}, {"{"}format{"}"}
                </small>
              </div>

              {/* Local Config */}
              {formData.destinationType === "local" && (
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
                    <label
                      htmlFor="local-path"
                      style={{
                        display: "block",
                        marginBottom: 4,
                        fontSize: 13,
                      }}
                    >
                      Path Template <span style={{ color: "#ef4444" }}>*</span>
                    </label>
                    <input
                      id="local-path"
                      type="text"
                      value={formData.localPath}
                      onChange={(e) => onChange({ localPath: e.target.value })}
                      placeholder="exports/{kind}/{job_id}.{format}"
                      style={{ width: "100%" }}
                    />
                  </div>
                </div>
              )}

              {/* Webhook Config */}
              {formData.destinationType === "webhook" && (
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
                    <label
                      htmlFor="webhook-url"
                      style={{
                        display: "block",
                        marginBottom: 4,
                        fontSize: 13,
                      }}
                    >
                      Webhook URL <span style={{ color: "#ef4444" }}>*</span>
                    </label>
                    <input
                      id="webhook-url"
                      type="url"
                      value={formData.webhookUrl}
                      onChange={(e) => onChange({ webhookUrl: e.target.value })}
                      placeholder="https://api.example.com/webhook"
                      style={{ width: "100%" }}
                    />
                  </div>
                </div>
              )}
            </div>

            {/* Result Transform */}
            <div
              style={{
                marginBottom: 24,
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
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
                  <div
                    style={{
                      color: "var(--muted)",
                      fontSize: 13,
                      marginTop: 4,
                    }}
                  >
                    Optionally project or reshape saved results with JMESPath or
                    JSONata before recurring export runs.
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
                    onClick={() => setShowTransformAssistant(true)}
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
                  Export shaping is active for this schedule. Clear the shape
                  before configuring a saved transform so recurring exports keep
                  one deterministic projection contract.
                </div>
              ) : (
                <>
                  <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
                    <div style={{ flex: "0 0 220px" }}>
                      <label
                        htmlFor="transform-language"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Transform language
                      </label>
                      <select
                        id="transform-language"
                        value={formData.transformLanguage}
                        onChange={(e) =>
                          onChange({
                            transformLanguage: e.target
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
                    <label
                      htmlFor="transform-expression"
                      style={{
                        display: "block",
                        marginBottom: 4,
                        fontSize: 13,
                      }}
                    >
                      Transform expression
                    </label>
                    <textarea
                      id="transform-expression"
                      value={formData.transformExpression}
                      onChange={(e) =>
                        onChange({ transformExpression: e.target.value })
                      }
                      rows={5}
                      placeholder={
                        formData.transformLanguage === "jmespath"
                          ? "{title: title, url: url, price: normalized.fields.price.values[0]}"
                          : '$.{"title": title, "url": url}'
                      }
                      style={{
                        width: "100%",
                        fontFamily: "monospace",
                        fontSize: 12,
                      }}
                    />
                    <small style={{ color: "var(--muted)" }}>
                      Leave empty to export the canonical saved results without
                      an additional transform.
                    </small>
                  </div>
                </>
              )}
            </div>

            {/* Export Shaping */}
            <div
              style={{
                marginBottom: 24,
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
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
                  <div
                    style={{
                      color: "var(--muted)",
                      fontSize: 13,
                      marginTop: 4,
                    }}
                  >
                    Customize recurring Markdown, CSV, and XLSX exports with a
                    bounded field-selection workflow.
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
                    onClick={() => setShowShapeAssistant(true)}
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
                  This schedule already has a saved transform. Clear the
                  transform before configuring export shaping so recurring
                  exports keep one deterministic projection path.
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
                    Use newline-separated field references. Add label overrides
                    as
                    <code style={{ marginLeft: 4 }}>field.key=Label</code>.
                  </div>

                  <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
                    <div style={{ flex: "1 1 260px" }}>
                      <label
                        htmlFor="shape-top-level-fields"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Top-level fields
                      </label>
                      <textarea
                        id="shape-top-level-fields"
                        value={formData.shapeTopLevelFields}
                        onChange={(e) =>
                          onChange({ shapeTopLevelFields: e.target.value })
                        }
                        rows={5}
                        placeholder="url&#10;title&#10;status"
                        style={{
                          width: "100%",
                          fontFamily: "monospace",
                          fontSize: 12,
                        }}
                      />
                    </div>
                    <div style={{ flex: "1 1 260px" }}>
                      <label
                        htmlFor="shape-summary-fields"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Summary fields
                      </label>
                      <textarea
                        id="shape-summary-fields"
                        value={formData.shapeSummaryFields}
                        onChange={(e) =>
                          onChange({ shapeSummaryFields: e.target.value })
                        }
                        rows={5}
                        placeholder="title&#10;field.price"
                        style={{
                          width: "100%",
                          fontFamily: "monospace",
                          fontSize: 12,
                        }}
                      />
                    </div>
                  </div>

                  <div
                    className="row"
                    style={{ gap: 16, flexWrap: "wrap", marginTop: 16 }}
                  >
                    <div style={{ flex: "1 1 260px" }}>
                      <label
                        htmlFor="shape-normalized-fields"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Normalized fields
                      </label>
                      <textarea
                        id="shape-normalized-fields"
                        value={formData.shapeNormalizedFields}
                        onChange={(e) =>
                          onChange({ shapeNormalizedFields: e.target.value })
                        }
                        rows={5}
                        placeholder="field.price&#10;field.plan"
                        style={{
                          width: "100%",
                          fontFamily: "monospace",
                          fontSize: 12,
                        }}
                      />
                    </div>
                    <div style={{ flex: "1 1 260px" }}>
                      <label
                        htmlFor="shape-evidence-fields"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Evidence fields
                      </label>
                      <textarea
                        id="shape-evidence-fields"
                        value={formData.shapeEvidenceFields}
                        onChange={(e) =>
                          onChange({ shapeEvidenceFields: e.target.value })
                        }
                        rows={5}
                        placeholder="evidence.url&#10;evidence.title"
                        style={{
                          width: "100%",
                          fontFamily: "monospace",
                          fontSize: 12,
                        }}
                      />
                    </div>
                  </div>

                  <div style={{ marginTop: 16 }}>
                    <label
                      htmlFor="shape-field-labels"
                      style={{
                        display: "block",
                        marginBottom: 4,
                        fontSize: 13,
                      }}
                    >
                      Field labels
                    </label>
                    <textarea
                      id="shape-field-labels"
                      value={formData.shapeFieldLabels}
                      onChange={(e) =>
                        onChange({ shapeFieldLabels: e.target.value })
                      }
                      rows={4}
                      placeholder="field.price=Price&#10;title=Page Title"
                      style={{
                        width: "100%",
                        fontFamily: "monospace",
                        fontSize: 12,
                      }}
                    />
                  </div>

                  <div
                    className="row"
                    style={{ gap: 16, flexWrap: "wrap", marginTop: 16 }}
                  >
                    <div style={{ flex: "1 1 180px" }}>
                      <label
                        htmlFor="shape-empty-value"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Empty value placeholder
                      </label>
                      <input
                        id="shape-empty-value"
                        type="text"
                        value={formData.shapeEmptyValue}
                        onChange={(e) =>
                          onChange({ shapeEmptyValue: e.target.value })
                        }
                        placeholder="—"
                        style={{ width: "100%" }}
                      />
                    </div>
                    <div style={{ flex: "1 1 180px" }}>
                      <label
                        htmlFor="shape-multi-value-join"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Multi-value join
                      </label>
                      <input
                        id="shape-multi-value-join"
                        type="text"
                        value={formData.shapeMultiValueJoin}
                        onChange={(e) =>
                          onChange({ shapeMultiValueJoin: e.target.value })
                        }
                        placeholder="; "
                        style={{ width: "100%" }}
                      />
                    </div>
                    <div style={{ flex: "1 1 220px" }}>
                      <label
                        htmlFor="shape-markdown-title"
                        style={{
                          display: "block",
                          marginBottom: 4,
                          fontSize: 13,
                        }}
                      >
                        Markdown title override
                      </label>
                      <input
                        id="shape-markdown-title"
                        type="text"
                        value={formData.shapeMarkdownTitle}
                        onChange={(e) =>
                          onChange({ shapeMarkdownTitle: e.target.value })
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
                  JSON and JSON Lines exports always ship the full structured
                  payload. Switch to Markdown, CSV, or XLSX to configure or
                  apply bounded export shaping.
                  {stagedShape ? (
                    <div style={{ marginTop: 8, color: "var(--text)" }}>
                      Shape fields are currently staged in the form but will not
                      be saved unless you switch back to a supported format.
                    </div>
                  ) : null}
                </div>
              )}
            </div>

            {/* Retry Config */}
            <div
              style={{
                marginBottom: 24,
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <h4 style={{ margin: "0 0 12px 0", fontSize: 14 }}>
                Retry Configuration
              </h4>
              <div className="row" style={{ gap: 16 }}>
                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="max-retries"
                    style={{ display: "block", marginBottom: 4, fontSize: 13 }}
                  >
                    Max Retries
                  </label>
                  <input
                    id="max-retries"
                    type="number"
                    min={0}
                    step={1}
                    value={formData.maxRetries}
                    onChange={(e) =>
                      onChange({
                        maxRetries: parseInt(e.target.value, 10) || 0,
                      })
                    }
                    style={{ width: "100%" }}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="base-delay"
                    style={{ display: "block", marginBottom: 4, fontSize: 13 }}
                  >
                    Base Delay (ms)
                  </label>
                  <input
                    id="base-delay"
                    type="number"
                    min={0}
                    step={100}
                    value={formData.baseDelayMs}
                    onChange={(e) =>
                      onChange({
                        baseDelayMs: parseInt(e.target.value, 10) || 0,
                      })
                    }
                    style={{ width: "100%" }}
                  />
                </div>
              </div>
            </div>

            <div className="row" style={{ gap: 8, justifyContent: "flex-end" }}>
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
          </form>
        </div>
      </div>

      {showTransformAssistant ? (
        <AIExportTransformAssistant
          isOpen={showTransformAssistant}
          onClose={() => setShowTransformAssistant(false)}
          aiStatus={aiStatus}
          currentTransform={currentTransform}
          onApplyTransform={(transform) =>
            onChange(transformConfigToFormData(transform))
          }
        />
      ) : null}

      {shapeSupported && showShapeAssistant ? (
        <AIExportShapeAssistant
          isOpen={showShapeAssistant}
          onClose={() => setShowShapeAssistant(false)}
          aiStatus={aiStatus}
          format={formData.format as "md" | "csv" | "xlsx"}
          currentShape={currentShape}
          onApplyShape={(shape) => onChange(shapeConfigToFormData(shape))}
        />
      ) : null}
    </>
  );
}
