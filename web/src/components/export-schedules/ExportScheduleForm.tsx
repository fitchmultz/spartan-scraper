/**
 * ExportScheduleForm Component
 *
 * Renders the create/edit form for export schedules in a modal dialog.
 * Includes all form fields: name, enabled, filters, export config,
 * destination-specific settings, and retry configuration.
 *
 * This component does NOT handle:
 * - API calls for saving export schedules (parent handles via onSubmit)
 * - Form state management (controlled via props)
 * - Modal visibility state
 *
 * @module components/export-schedules/ExportScheduleForm
 */

import type { ExportScheduleFormProps } from "../../types/export-schedule";

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
  { value: "parquet", label: "Parquet" },
  { value: "har", label: "HAR" },
  { value: "pdf", label: "PDF" },
];

const DESTINATION_OPTIONS = [
  { value: "local", label: "Local File" },
  { value: "webhook", label: "Webhook" },
  { value: "s3", label: "Amazon S3" },
  { value: "gcs", label: "Google Cloud Storage" },
  { value: "azure", label: "Azure Blob Storage" },
];

const CLOUD_PROVIDER_OPTIONS = [
  { value: "s3", label: "Amazon S3" },
  { value: "gcs", label: "Google Cloud Storage" },
  { value: "azure", label: "Azure Blob Storage" },
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
}: ExportScheduleFormProps) {
  const isCloudDestination = ["s3", "gcs", "azure"].includes(
    formData.destinationType,
  );

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
          maxWidth: 700,
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
            <h4 style={{ margin: "0 0 12px 0", fontSize: 14 }}>Job Filters</h4>

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
                placeholder="{job_id}.{format}"
                style={{ width: "100%" }}
              />
              <small style={{ color: "var(--muted)" }}>
                Variables: {"{"}job_id{"}"}, {"{"}timestamp{"}"}, {"{"}kind{"}"}
                , {"{"}format{"}"}
              </small>
            </div>

            {/* Cloud Config */}
            {isCloudDestination && (
              <div
                style={{
                  marginTop: 16,
                  padding: 16,
                  backgroundColor: "var(--bg)",
                  borderRadius: 6,
                }}
              >
                <h5 style={{ margin: "0 0 12px 0", fontSize: 13 }}>
                  Cloud Configuration
                </h5>

                <div style={{ marginBottom: 12 }}>
                  <label
                    htmlFor="cloud-provider"
                    style={{ display: "block", marginBottom: 4, fontSize: 13 }}
                  >
                    Provider
                  </label>
                  <select
                    id="cloud-provider"
                    value={formData.cloudProvider}
                    onChange={(e) =>
                      onChange({
                        cloudProvider: e.target
                          .value as typeof formData.cloudProvider,
                      })
                    }
                    style={{ width: "100%" }}
                  >
                    {CLOUD_PROVIDER_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <label
                    htmlFor="cloud-bucket"
                    style={{ display: "block", marginBottom: 4, fontSize: 13 }}
                  >
                    Bucket/Container <span style={{ color: "#ef4444" }}>*</span>
                  </label>
                  <input
                    id="cloud-bucket"
                    type="text"
                    value={formData.cloudBucket}
                    onChange={(e) => onChange({ cloudBucket: e.target.value })}
                    placeholder="my-bucket"
                    style={{ width: "100%" }}
                  />
                </div>

                <div className="row" style={{ gap: 12 }}>
                  <div style={{ flex: 1 }}>
                    <label
                      htmlFor="cloud-region"
                      style={{
                        display: "block",
                        marginBottom: 4,
                        fontSize: 13,
                      }}
                    >
                      Region
                    </label>
                    <input
                      id="cloud-region"
                      type="text"
                      value={formData.cloudRegion}
                      onChange={(e) =>
                        onChange({ cloudRegion: e.target.value })
                      }
                      placeholder="us-east-1"
                      style={{ width: "100%" }}
                    />
                  </div>
                  <div style={{ flex: 1 }}>
                    <label
                      htmlFor="cloud-path"
                      style={{
                        display: "block",
                        marginBottom: 4,
                        fontSize: 13,
                      }}
                    >
                      Path
                    </label>
                    <input
                      id="cloud-path"
                      type="text"
                      value={formData.cloudPath}
                      onChange={(e) => onChange({ cloudPath: e.target.value })}
                      placeholder="exports/"
                      style={{ width: "100%" }}
                    />
                  </div>
                </div>
              </div>
            )}

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
                    style={{ display: "block", marginBottom: 4, fontSize: 13 }}
                  >
                    Path Template <span style={{ color: "#ef4444" }}>*</span>
                  </label>
                  <input
                    id="local-path"
                    type="text"
                    value={formData.localPath}
                    onChange={(e) => onChange({ localPath: e.target.value })}
                    placeholder="/data/exports/{job_id}.{format}"
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
                    style={{ display: "block", marginBottom: 4, fontSize: 13 }}
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
  );
}
