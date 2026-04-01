/**
 * Purpose: Provide extracted presentational sections for the retention status surface.
 * Responsibilities: Render preview feedback, capability banners, retention summaries, cleanup controls, and cleanup result notices.
 * Scope: Retention panel presentation only; derived helper logic stays in `retentionStatusPanelUtils.ts` and API orchestration stays in `RetentionStatusPanel.tsx`.
 * Usage: Import from `RetentionStatusPanel.tsx` to keep the main component focused on fetch/mutation state.
 * Invariants/Assumptions: Existing operator copy, labels, and cleanup safeguards remain stable for current workflows and tests.
 */

import type {
  RecommendedAction,
  RetentionCleanupResponse,
  RetentionStatusResponse,
} from "../api";
import { ActionEmptyState } from "./ActionEmptyState";
import { CapabilityActionList } from "./CapabilityActionList";
import { CapabilityLoadErrorState } from "./CapabilityLoadErrorState";
import type {
  CleanupActivity,
  ResolvedRetentionCapability,
  RetentionLoadErrorState,
} from "./retentionStatusPanelUtils";

interface CleanupPreviewFeedbackProps {
  activity: CleanupActivity | null;
  loading: boolean;
  error: string | null;
  result: RetentionCleanupResponse | null;
}

interface StatusCardProps {
  label: string;
  value: string | number;
  unit?: string;
  highlight?: "normal" | "warning" | "danger";
}

function formatCleanupScope(activity: CleanupActivity) {
  return {
    scopeLabel: activity.kind
      ? `${activity.kind[0].toUpperCase()}${activity.kind.slice(1)} jobs only`
      : "All job kinds",
    ageLabel:
      activity.olderThanDays !== null
        ? `Older than ${activity.olderThanDays} day${activity.olderThanDays === 1 ? "" : "s"} for this run`
        : "Current retention policy",
    usesDefaultScope: activity.kind === "" && activity.olderThanDays === null,
  };
}

export function CleanupPreviewFeedback({
  activity,
  loading,
  error,
  result,
}: CleanupPreviewFeedbackProps) {
  if (!activity?.dryRun) {
    return null;
  }

  const scope = formatCleanupScope(activity);
  const hasNoMatches =
    !loading &&
    !error &&
    result &&
    result.jobsDeleted === 0 &&
    result.crawlStatesDeleted === 0;

  let title = "Preview complete";
  let tone = "info";
  let message = "Review the scoped preview summary below before deleting data.";

  if (loading) {
    title = "Previewing cleanup";
    message =
      "Running a dry-run preview now. Results will appear here and in the summary below.";
  } else if (error) {
    title = "Preview failed";
    tone = "warning";
    message = error;
  } else if (hasNoMatches) {
    message = "No jobs or crawl states matched this preview.";
  }

  return (
    <div
      className={`retention-notice retention-notice--${tone} retention-preview-feedback`}
      role="status"
      aria-live="polite"
    >
      <h4 className="retention-section-title retention-notice__title">
        {title}
      </h4>
      <p className="retention-notice__copy">{message}</p>
      <div className="retention-preview-feedback__meta">
        <span>
          <strong>Scope:</strong> {scope.scopeLabel}
        </span>
        <span>
          <strong>Age:</strong> {scope.ageLabel}
        </span>
        {activity.force ? (
          <span>
            <strong>Retention:</strong> Dry-run preview ignores the disabled
            cleanup gate so you can see the blast radius safely.
          </span>
        ) : null}
      </div>
      {scope.usesDefaultScope ? (
        <div className="retention-notice__detail">
          Blank filters preview all job kinds using the current retention
          policy.
        </div>
      ) : null}
    </div>
  );
}

function StatusCard({
  label,
  value,
  unit,
  highlight = "normal",
}: StatusCardProps) {
  return (
    <div
      className={`retention-status-card retention-status-card--${highlight}`}
    >
      <div className="retention-status-card__label">{label}</div>
      <div className="retention-status-card__value">
        {value}
        {unit ? (
          <span className="retention-status-card__unit">{unit}</span>
        ) : null}
      </div>
    </div>
  );
}

export function RetentionCapabilitySurface({
  loading,
  error,
  status,
  capability,
  cleanupActivity,
  cleanupLoading,
  cleanupResult,
  loadErrorState,
  loadErrorActions,
  onNavigate,
  onRefresh,
  onPreviewCleanup,
  onCreateJob,
  onOpenAutomation,
}: {
  loading: boolean;
  error: string | null;
  status: RetentionStatusResponse | null;
  capability: ResolvedRetentionCapability;
  cleanupActivity: CleanupActivity | null;
  cleanupLoading: boolean;
  cleanupResult: RetentionCleanupResponse | null;
  loadErrorState: RetentionLoadErrorState | null;
  loadErrorActions: RecommendedAction[];
  onNavigate: (path: string) => void;
  onRefresh: () => Promise<unknown>;
  onPreviewCleanup: () => void;
  onCreateJob: () => void;
  onOpenAutomation: () => void;
}) {
  const showLoadErrorState = Boolean(error && !loading && !status);

  if (loading && !status) {
    return <div>Loading retention status...</div>;
  }

  if (showLoadErrorState && loadErrorState && error) {
    return (
      <CapabilityLoadErrorState
        eyebrow={loadErrorState.eyebrow}
        title={loadErrorState.title}
        description={loadErrorState.description}
        error={error}
        actions={loadErrorActions}
        onNavigate={onNavigate}
        onRefresh={onRefresh}
      />
    );
  }

  if (!status) {
    return null;
  }

  if (capability.status === "disabled") {
    return (
      <ActionEmptyState
        eyebrow="Optional subsystem"
        title={capability.title || "Automatic retention stays off by default"}
        description={capability.message}
        actions={[
          { label: "Preview cleanup", onClick: onPreviewCleanup },
          {
            label: "Create job",
            onClick: onCreateJob,
            tone: "secondary",
          },
        ]}
      >
        {cleanupActivity?.origin === "banner" ? (
          <CleanupPreviewFeedback
            activity={cleanupActivity}
            loading={cleanupLoading}
            error={error}
            result={cleanupResult}
          />
        ) : null}
        <CapabilityActionList
          actions={capability.actions}
          onNavigate={onNavigate}
          onRefresh={onRefresh}
        />
      </ActionEmptyState>
    );
  }

  if (
    ["warning", "danger", "degraded", "error", "setup_required"].includes(
      capability.status,
    ) &&
    capability.message
  ) {
    return (
      <div
        className={`retention-notice ${
          capability.status === "danger" || capability.status === "error"
            ? "retention-notice--warning"
            : "retention-notice--info"
        }`}
      >
        <h4 className="retention-section-title">{capability.title}</h4>
        <p className="retention-notice__copy">{capability.message}</p>
        <div className="retention-notice__actions">
          {capability.status === "warning" || capability.status === "danger" ? (
            <>
              <button
                type="button"
                className="secondary"
                onClick={onPreviewCleanup}
                disabled={cleanupLoading}
              >
                {cleanupLoading ? "Running..." : "Preview cleanup"}
              </button>
              <button
                type="button"
                className="secondary"
                onClick={onOpenAutomation}
              >
                Open automation
              </button>
            </>
          ) : (
            <button
              type="button"
              className="secondary"
              onClick={() => void onRefresh()}
            >
              Refresh status
            </button>
          )}
        </div>

        {cleanupActivity?.origin === "banner" &&
        (capability.status === "warning" || capability.status === "danger") ? (
          <CleanupPreviewFeedback
            activity={cleanupActivity}
            loading={cleanupLoading}
            error={error}
            result={cleanupResult}
          />
        ) : null}

        <CapabilityActionList
          actions={capability.actions}
          onNavigate={onNavigate}
          onRefresh={onRefresh}
        />
      </div>
    );
  }

  return null;
}

export function RetentionStatusSummary({
  status,
  jobsHighlight,
  storageHighlight,
}: {
  status: RetentionStatusResponse;
  jobsHighlight: "normal" | "warning" | "danger";
  storageHighlight: "normal" | "warning" | "danger";
}) {
  return (
    <>
      <div className="retention-status-grid">
        <StatusCard
          label="Retention Enabled"
          value={status.enabled ? "Yes" : "No"}
          highlight={status.enabled ? "normal" : "warning"}
        />
        <StatusCard
          label="Total Jobs"
          value={status.totalJobs.toLocaleString()}
          highlight={jobsHighlight}
        />
        <StatusCard
          label="Storage Used"
          value={
            status.storageUsedMB >= 1024
              ? (status.storageUsedMB / 1024).toFixed(2)
              : status.storageUsedMB
          }
          unit={status.storageUsedMB >= 1024 ? "GB" : "MB"}
          highlight={storageHighlight}
        />
        <StatusCard
          label="Jobs Eligible for Cleanup"
          value={status.jobsEligible.toLocaleString()}
          highlight={status.jobsEligible > 0 ? "warning" : "normal"}
        />
      </div>

      <div className="retention-config-card">
        <h4 className="retention-section-title">Configuration</h4>
        <div className="retention-config-grid">
          <div className="retention-metric">
            <span className="retention-metric__label">Job Retention</span>
            <span className="retention-metric__value">
              {status.jobRetentionDays > 0
                ? `${status.jobRetentionDays} days`
                : "Unlimited"}
            </span>
          </div>
          <div className="retention-metric">
            <span className="retention-metric__label">
              Crawl State Retention
            </span>
            <span className="retention-metric__value">
              {status.crawlStateDays > 0
                ? `${status.crawlStateDays} days`
                : "Unlimited"}
            </span>
          </div>
          <div className="retention-metric">
            <span className="retention-metric__label">Max Jobs</span>
            <span className="retention-metric__value">
              {status.maxJobs > 0
                ? status.maxJobs.toLocaleString()
                : "Unlimited"}
            </span>
          </div>
          <div className="retention-metric">
            <span className="retention-metric__label">Max Storage</span>
            <span className="retention-metric__value">
              {status.maxStorageGB > 0
                ? `${status.maxStorageGB} GB`
                : "Unlimited"}
            </span>
          </div>
        </div>
      </div>
    </>
  );
}

export function RetentionCleanupControls({
  status,
  dryRun,
  kind,
  olderThan,
  loading,
  cleanupLoading,
  showConfirm,
  error,
  cleanupActivity,
  cleanupResult,
  onChangeDryRun,
  onChangeKind,
  onChangeOlderThan,
  onPreviewOrRun,
  onRefresh,
  onConfirmDelete,
  onCancelConfirm,
}: {
  status: RetentionStatusResponse;
  dryRun: boolean;
  kind: CleanupActivity["kind"];
  olderThan: string;
  loading: boolean;
  cleanupLoading: boolean;
  showConfirm: boolean;
  error: string | null;
  cleanupActivity: CleanupActivity | null;
  cleanupResult: RetentionCleanupResponse | null;
  onChangeDryRun: (value: boolean) => void;
  onChangeKind: (value: CleanupActivity["kind"]) => void;
  onChangeOlderThan: (value: string) => void;
  onPreviewOrRun: () => void;
  onRefresh: () => Promise<unknown>;
  onConfirmDelete: () => void;
  onCancelConfirm: () => void;
}) {
  return (
    <>
      <div className="retention-controls">
        <h4 className="retention-section-title">Cleanup Controls</h4>

        <div className="retention-controls__toggle-row">
          <label className="retention-controls__toggle">
            <input
              type="checkbox"
              checked={dryRun}
              onChange={(event) => onChangeDryRun(event.target.checked)}
            />
            <span>Dry-run mode (preview only, no actual deletions)</span>
          </label>
        </div>

        <p className="retention-controls__hint">
          Leave both filters blank to preview all job kinds using the current
          retention policy.
          {!status.enabled
            ? " Dry-run previews still run while automatic retention is off."
            : ""}
        </p>

        {cleanupActivity?.origin === "controls" ? (
          <CleanupPreviewFeedback
            activity={cleanupActivity}
            loading={cleanupLoading}
            error={error}
            result={cleanupResult}
          />
        ) : null}

        <div className="retention-controls__grid">
          <div>
            <label htmlFor="kind-select" className="retention-controls__label">
              Job Kind (optional)
            </label>
            <select
              id="kind-select"
              value={kind}
              onChange={(event) =>
                onChangeKind(
                  event.target.value as "" | "scrape" | "crawl" | "research",
                )
              }
            >
              <option value="">All kinds</option>
              <option value="scrape">Scrape</option>
              <option value="crawl">Crawl</option>
              <option value="research">Research</option>
            </select>
          </div>
          <div>
            <label
              htmlFor="older-than-input"
              className="retention-controls__label"
            >
              Older Than (days, optional)
            </label>
            <input
              id="older-than-input"
              type="number"
              min="1"
              value={olderThan}
              onChange={(event) => onChangeOlderThan(event.target.value)}
              placeholder="Use config default"
            />
          </div>
        </div>

        <div className="retention-controls__actions">
          <button
            type="button"
            onClick={onPreviewOrRun}
            disabled={cleanupLoading || (!status.enabled && !dryRun)}
            className={dryRun ? "secondary" : "primary"}
          >
            {cleanupLoading
              ? "Running..."
              : dryRun
                ? "Preview Cleanup"
                : "Run Cleanup"}
          </button>
          <button
            type="button"
            onClick={() => void onRefresh()}
            disabled={loading}
            className="secondary"
          >
            {loading ? "Refreshing..." : "Refresh Status"}
          </button>
          {!status.enabled && !dryRun ? (
            <span className="retention-controls__disabled-message">
              Automatic retention is off. Turn on dry-run to preview what
              cleanup would do.
            </span>
          ) : null}
        </div>
      </div>

      {showConfirm ? (
        <div className="retention-notice retention-notice--warning">
          <p className="retention-notice__copy">
            <strong>Warning:</strong> This will permanently delete jobs and
            their artifacts. This action cannot be undone.
          </p>
          <div className="retention-notice__actions">
            <button
              type="button"
              onClick={onConfirmDelete}
              disabled={cleanupLoading}
              className="retention-action-button retention-action-button--danger"
            >
              {cleanupLoading ? "Running..." : "Confirm Delete"}
            </button>
            <button
              type="button"
              onClick={onCancelConfirm}
              disabled={cleanupLoading}
              className="secondary"
            >
              Cancel
            </button>
          </div>
        </div>
      ) : null}
    </>
  );
}

export function CleanupResultNotice({
  cleanupResult,
}: {
  cleanupResult: RetentionCleanupResponse | null;
}) {
  if (!cleanupResult) {
    return null;
  }

  return (
    <div
      className={`retention-notice ${
        cleanupResult.dryRun
          ? "retention-notice--info"
          : "retention-notice--success"
      }`}
    >
      <h4 className="retention-section-title retention-notice__title">
        {cleanupResult.dryRun ? "Dry-Run Preview Results" : "Cleanup Complete"}
      </h4>
      <div className="retention-results-grid">
        <div className="retention-metric">
          <span className="retention-metric__label">
            Jobs {cleanupResult.dryRun ? "Would Delete" : "Deleted"}:{" "}
          </span>
          <strong className="retention-metric__value">
            {cleanupResult.jobsDeleted.toLocaleString()}
          </strong>
        </div>
        <div className="retention-metric">
          <span className="retention-metric__label">Jobs Attempted</span>
          <strong className="retention-metric__value">
            {cleanupResult.jobsAttempted.toLocaleString()}
          </strong>
        </div>
        <div className="retention-metric">
          <span className="retention-metric__label">
            Crawl States {cleanupResult.dryRun ? "Would Delete" : "Deleted"}:{" "}
          </span>
          <strong className="retention-metric__value">
            {cleanupResult.crawlStatesDeleted.toLocaleString()}
          </strong>
        </div>
        <div className="retention-metric">
          <span className="retention-metric__label">
            Space {cleanupResult.dryRun ? "Would Reclaim" : "Reclaimed"}:{" "}
          </span>
          <strong className="retention-metric__value">
            {cleanupResult.spaceReclaimedMB >= 1024
              ? `${(cleanupResult.spaceReclaimedMB / 1024).toFixed(2)} GB`
              : `${cleanupResult.spaceReclaimedMB} MB`}
          </strong>
        </div>
        <div className="retention-metric">
          <span className="retention-metric__label">Duration</span>
          <strong className="retention-metric__value">
            {cleanupResult.durationMs}ms
          </strong>
        </div>
      </div>
      {cleanupResult.failedJobIDs && cleanupResult.failedJobIDs.length > 0 ? (
        <div className="retention-notice__detail">
          <span className="retention-notice__danger-text">
            Warning: {cleanupResult.failedJobIDs.length} job(s) had artifact
            deletion failures
          </span>
        </div>
      ) : null}
      {cleanupResult.errors && cleanupResult.errors.length > 0 ? (
        <div className="retention-notice__detail">
          <span className="retention-notice__danger-text">
            Errors ({cleanupResult.errors.length}):
          </span>
          <ul className="retention-notice__list">
            {cleanupResult.errors.map((cleanupError) => (
              <li key={cleanupError}>{cleanupError}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
