/**
 * Purpose: Render the guided wizard preflight summary before a job is submitted.
 * Responsibilities: Summarize the current job configuration, highlight key execution/extraction decisions, and surface non-blocking warnings.
 * Scope: Guided job wizard review step only.
 * Usage: Render from `JobSubmissionContainer` while guided mode is on the `review` step.
 * Invariants/Assumptions: Review summaries read from the current preset-style config snapshot and warnings are informative rather than submission-blocking.
 */

import type { JobType, PresetConfig } from "../../../types/presets";

function summarizeValue(value: unknown): string {
  if (typeof value === "boolean") {
    return value ? "Enabled" : "Disabled";
  }

  if (Array.isArray(value)) {
    return value.length > 0 ? value.join(", ") : "None";
  }

  if (value === "" || value == null) {
    return "Not set";
  }

  return String(value);
}

interface ReviewStepProps {
  activeTab: JobType;
  config: PresetConfig;
  warnings: string[];
}

export function ReviewStep({ activeTab, config, warnings }: ReviewStepProps) {
  const primaryTarget = activeTab === "research" ? config.query : config.url;
  const extractionSummary = config.extractTemplate
    ? config.extractTemplate
    : config.aiExtractEnabled
      ? "AI extraction"
      : "No template or AI extraction";

  return (
    <section className="panel job-wizard__panel">
      <div className="job-wizard__panel-header">
        <div className="job-workflow__eyebrow">Review & Submit</div>
        <h2>Review the configuration before launch</h2>
        <p>
          This is the preflight checkpoint: confirm the target, runtime
          behavior, extraction plan, and any risky flags.
        </p>
      </div>

      <div className="job-wizard__summary-grid">
        <div className="job-wizard__summary-card">
          <span>Job type</span>
          <strong>{activeTab}</strong>
        </div>
        <div className="job-wizard__summary-card">
          <span>{activeTab === "research" ? "Query" : "Target"}</span>
          <strong>{summarizeValue(primaryTarget)}</strong>
        </div>
        <div className="job-wizard__summary-card">
          <span>Browser mode</span>
          <strong>
            {config.headless ? "Headless browser" : "HTTP / non-headless"}
          </strong>
        </div>
        <div className="job-wizard__summary-card">
          <span>Extraction</span>
          <strong>{extractionSummary}</strong>
        </div>
      </div>

      <div className="job-wizard__review-list">
        <div>
          <span>Timeout</span>
          <strong>{summarizeValue(config.timeoutSeconds)}s</strong>
        </div>
        <div>
          <span>Playwright</span>
          <strong>{summarizeValue(config.usePlaywright)}</strong>
        </div>
        <div>
          <span>Template validation</span>
          <strong>{summarizeValue(config.extractValidate)}</strong>
        </div>
        <div>
          <span>Webhook</span>
          <strong>
            {config.webhookUrl ? config.webhookUrl : "Not configured"}
          </strong>
        </div>
      </div>

      {warnings.length > 0 ? (
        <div className="job-wizard__warning-summary" role="status">
          <strong>Warnings</strong>
          <ul>
            {warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
