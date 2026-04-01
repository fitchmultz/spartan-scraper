/**
 * Purpose: Render the shell, basic metadata, filters, and destination configuration sections for export schedule authoring.
 * Responsibilities: Present the dialog frame and the core scheduling filters/configuration fields while preserving stable labels and layout.
 * Scope: Export schedule form presentation only; AI assistant state and mutation orchestration stay in `ExportScheduleForm`.
 * Usage: Compose these sections from `ExportScheduleForm` through the local barrel.
 * Invariants/Assumptions: Field ids, operator copy, and controlled-form behavior remain stable for current workflows and tests.
 */

import type { ReactNode } from "react";

import type { ExportScheduleFormProps } from "../../types/export-schedule";
import { AIUnavailableNotice } from "../ai-assistant";
import { PromotionDraftNotice } from "../promotion/PromotionDraftNotice";
import {
  DESTINATION_OPTIONS,
  fieldLabelStyle,
  FORMAT_OPTIONS,
  JOB_KIND_OPTIONS,
  JOB_STATUS_OPTIONS,
  sectionStyle,
  sectionTitleStyle,
} from "./exportScheduleFormShared";

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
