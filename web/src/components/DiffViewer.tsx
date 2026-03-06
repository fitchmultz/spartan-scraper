/**
 * Diff Viewer Component
 *
 * Displays side-by-side comparison of two job results with color-coded changes.
 * Shows added, removed, and modified items with detailed field-level diffs for
 * crawl results. Supports filtering by change type and summary statistics.
 *
 * @module DiffViewer
 */
import { useMemo, useState } from "react";
import type {
  CrawlDiffResult,
  ResearchDiffResult,
  FieldChange,
} from "../lib/diff-utils";
import { getCrawlDiffStats, getResearchDiffStats } from "../lib/diff-utils";
import type { CrawlResultItem, Job } from "../types";

interface DiffViewerProps {
  /** Base job ("before" state) */
  baseJob: Job | null;
  /** Comparison job ("after" state) */
  compareJob: Job | null;
  /** Diff result data */
  diffResult: CrawlDiffResult | ResearchDiffResult | null;
  /** Whether diff is loading */
  isLoading: boolean;
  /** Error message if diff failed */
  error: string | null;
  /** Callback to close diff view */
  onClose: () => void;
}

type ChangeType = "added" | "removed" | "modified" | "unchanged";

/**
 * Format a field value for display.
 */
function formatValue(value: unknown): string {
  if (value === undefined) return "undefined";
  if (value === null) return "null";
  if (typeof value === "string") return value;
  if (typeof value === "number") return String(value);
  if (typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return `[${value.length} items]`;
  if (typeof value === "object") return "{...}";
  return String(value);
}

/**
 * Truncate text to a maximum length.
 */
function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength)}...`;
}

/**
 * Stat card component for diff summary.
 */
function StatCard({
  label,
  count,
  type,
}: {
  label: string;
  count: number;
  type: ChangeType | "total";
}) {
  const className = `diff-stat-card ${type}`;
  return (
    <div className={className}>
      <div className="diff-stat-count">{count}</div>
      <div className="diff-stat-label">{label}</div>
    </div>
  );
}

/**
 * Field change row component.
 */
function FieldChangeRow({ change }: { change: FieldChange }) {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <div className="diff-field-change">
      <button
        type="button"
        className="diff-field-header"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <span className="diff-field-name">{change.field}</span>
        <span className="diff-field-toggle">{isExpanded ? "−" : "+"}</span>
      </button>
      {isExpanded && (
        <div className="diff-field-values">
          <div className="diff-field-old">
            <span className="diff-field-label">Before:</span>
            <pre>{truncate(formatValue(change.oldValue), 200)}</pre>
          </div>
          <div className="diff-field-new">
            <span className="diff-field-label">After:</span>
            <pre>{truncate(formatValue(change.newValue), 200)}</pre>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Crawl result diff item component.
 */
function CrawlDiffItem({
  item,
  type,
  changes,
}: {
  item: CrawlResultItem;
  type: ChangeType;
  changes?: FieldChange[];
}) {
  const [showDetails, setShowDetails] = useState(false);

  return (
    <div className={`diff-item ${type}`}>
      <div className="diff-item-header">
        <span className={`diff-item-badge ${type}`}>
          {type === "added" && "+"}
          {type === "removed" && "−"}
          {type === "modified" && "~"}
          {type === "unchanged" && "="}
        </span>
        <span className="diff-item-url" title={item.url}>
          {truncate(item.url, 60)}
        </span>
        <span className={`badge ${getStatusBadgeClass(item.status)}`}>
          {item.status}
        </span>
        {changes && changes.length > 0 && (
          <button
            type="button"
            className="diff-item-toggle"
            onClick={() => setShowDetails(!showDetails)}
          >
            {showDetails ? "Hide" : "Show"} {changes.length} changes
          </button>
        )}
      </div>
      {item.title && (
        <div className="diff-item-title">{truncate(item.title, 80)}</div>
      )}
      {showDetails && changes && changes.length > 0 && (
        <div className="diff-item-changes">
          {changes.map((change) => (
            <FieldChangeRow key={change.field} change={change} />
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Get CSS class for HTTP status code.
 */
function getStatusBadgeClass(status: number): string {
  if (status >= 200 && status < 300) return "success";
  if (status >= 400) return "failed";
  return "running";
}

/**
 * Crawl diff view component.
 */
function CrawlDiffView({
  diff,
  visibleTypes,
}: {
  diff: CrawlDiffResult;
  visibleTypes: Set<ChangeType>;
}) {
  const items: Array<{
    item: CrawlResultItem;
    type: ChangeType;
    changes?: FieldChange[];
  }> = [];

  if (visibleTypes.has("added")) {
    for (const item of diff.added) {
      items.push({ item, type: "added" });
    }
  }

  if (visibleTypes.has("removed")) {
    for (const item of diff.removed) {
      items.push({ item, type: "removed" });
    }
  }

  if (visibleTypes.has("modified")) {
    for (const mod of diff.modified) {
      items.push({
        item: mod.after,
        type: "modified",
        changes: mod.changes,
      });
    }
  }

  if (visibleTypes.has("unchanged")) {
    for (const item of diff.unchanged) {
      items.push({ item, type: "unchanged" });
    }
  }

  if (items.length === 0) {
    return (
      <div className="diff-empty">
        No items match the selected filter criteria.
      </div>
    );
  }

  return (
    <div className="diff-list">
      {items.map((entry) => (
        <CrawlDiffItem
          key={`${entry.type}-${entry.item.url}-${entry.changes?.map((change) => change.field).join(",") || "unchanged"}`}
          item={entry.item}
          type={entry.type}
          changes={entry.changes}
        />
      ))}
    </div>
  );
}

/**
 * Research diff view component.
 */
function ResearchDiffView({ diff }: { diff: ResearchDiffResult }) {
  const stats = getResearchDiffStats(diff);

  return (
    <div className="diff-research">
      {/* Summary Section */}
      {diff.summaryChanges && (
        <div className="diff-section">
          <h4>Summary</h4>
          <div className="diff-summary-changes">
            <div className="diff-old">
              <span className="diff-label">Before:</span>
              <p>{diff.summaryChanges.oldValue || "(none)"}</p>
            </div>
            <div className="diff-new">
              <span className="diff-label">After:</span>
              <p>{diff.summaryChanges.newValue || "(none)"}</p>
            </div>
          </div>
        </div>
      )}

      {/* Confidence Section */}
      {diff.confidenceChanges && (
        <div className="diff-section">
          <h4>Confidence Score</h4>
          <div className="diff-confidence-changes">
            <div className="diff-old">
              <span className="diff-label">Before:</span>
              <span className="diff-score">
                {diff.confidenceChanges.oldValue?.toFixed(2) || "N/A"}
              </span>
            </div>
            <div className="diff-arrow">→</div>
            <div className="diff-new">
              <span className="diff-label">After:</span>
              <span className="diff-score">
                {diff.confidenceChanges.newValue?.toFixed(2) || "N/A"}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Evidence Section */}
      {(stats.evidenceAdded > 0 ||
        stats.evidenceRemoved > 0 ||
        stats.evidenceModified > 0) && (
        <div className="diff-section">
          <h4>Evidence Changes</h4>
          <div className="diff-stats-row">
            <StatCard label="Added" count={stats.evidenceAdded} type="added" />
            <StatCard
              label="Removed"
              count={stats.evidenceRemoved}
              type="removed"
            />
            <StatCard
              label="Modified"
              count={stats.evidenceModified}
              type="modified"
            />
          </div>
        </div>
      )}

      {/* Clusters Section */}
      {(stats.clustersAdded > 0 ||
        stats.clustersRemoved > 0 ||
        stats.clustersModified > 0) && (
        <div className="diff-section">
          <h4>Cluster Changes</h4>
          <div className="diff-stats-row">
            <StatCard label="Added" count={stats.clustersAdded} type="added" />
            <StatCard
              label="Removed"
              count={stats.clustersRemoved}
              type="removed"
            />
            <StatCard
              label="Modified"
              count={stats.clustersModified}
              type="modified"
            />
          </div>
        </div>
      )}

      {/* Citations Section */}
      {(stats.citationsAdded > 0 || stats.citationsRemoved > 0) && (
        <div className="diff-section">
          <h4>Citation Changes</h4>
          <div className="diff-stats-row">
            <StatCard label="Added" count={stats.citationsAdded} type="added" />
            <StatCard
              label="Removed"
              count={stats.citationsRemoved}
              type="removed"
            />
          </div>
        </div>
      )}

      {/* No changes message */}
      {!diff.summaryChanges &&
        !diff.confidenceChanges &&
        stats.evidenceAdded === 0 &&
        stats.evidenceRemoved === 0 &&
        stats.evidenceModified === 0 &&
        stats.clustersAdded === 0 &&
        stats.clustersRemoved === 0 &&
        stats.clustersModified === 0 &&
        stats.citationsAdded === 0 &&
        stats.citationsRemoved === 0 && (
          <div className="diff-empty">
            No changes detected between versions.
          </div>
        )}
    </div>
  );
}

/**
 * Main DiffViewer component.
 *
 * Displays a comparison between two job results with filtering and statistics.
 */
export function DiffViewer({
  baseJob,
  compareJob,
  diffResult,
  isLoading,
  error,
  onClose,
}: DiffViewerProps) {
  const [visibleTypes, setVisibleTypes] = useState<Set<ChangeType>>(
    new Set(["added", "removed", "modified"]),
  );

  const isCrawlDiff = useMemo(() => {
    if (!diffResult) return false;
    return "added" in diffResult && Array.isArray(diffResult.added);
  }, [diffResult]);

  const stats = useMemo(() => {
    if (!diffResult) return null;
    if (isCrawlDiff) {
      return getCrawlDiffStats(diffResult as CrawlDiffResult);
    }
    return null;
  }, [diffResult, isCrawlDiff]);

  const toggleType = (type: ChangeType) => {
    setVisibleTypes((prev) => {
      const next = new Set(prev);
      if (next.has(type)) {
        next.delete(type);
      } else {
        next.add(type);
      }
      return next;
    });
  };

  if (isLoading) {
    return (
      <div className="diff-viewer loading">
        <div className="diff-loading">Loading comparison...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="diff-viewer error">
        <div className="diff-error">{error}</div>
        <button type="button" className="secondary" onClick={onClose}>
          Close
        </button>
      </div>
    );
  }

  if (!diffResult || !baseJob || !compareJob) {
    return (
      <div className="diff-viewer empty">
        <div className="diff-empty-message">
          Select two jobs to compare their results.
        </div>
      </div>
    );
  }

  return (
    <div className="diff-viewer">
      <div className="diff-header">
        <h3>Comparing Results</h3>
        <button type="button" className="secondary" onClick={onClose}>
          Close
        </button>
      </div>

      <div className="diff-jobs-info">
        <div className="diff-job-base">
          <span className="diff-label">Base:</span>
          <span className="diff-job-id">{baseJob.id}</span>
          <span className={`badge ${baseJob.status}`}>{baseJob.status}</span>
        </div>
        <div className="diff-arrow">→</div>
        <div className="diff-job-compare">
          <span className="diff-label">Compare:</span>
          <span className="diff-job-id">{compareJob.id}</span>
          <span className={`badge ${compareJob.status}`}>
            {compareJob.status}
          </span>
        </div>
      </div>

      {stats && (
        <div className="diff-stats">
          <StatCard label="Added" count={stats.added} type="added" />
          <StatCard label="Removed" count={stats.removed} type="removed" />
          <StatCard label="Modified" count={stats.modified} type="modified" />
          <StatCard label="Unchanged" count={stats.unchanged} type="total" />
        </div>
      )}

      {isCrawlDiff && (
        <div className="diff-filters">
          <span className="diff-filter-label">Show:</span>
          {(["added", "removed", "modified", "unchanged"] as ChangeType[]).map(
            (type) => (
              <label key={type} className="diff-filter-checkbox">
                <input
                  type="checkbox"
                  checked={visibleTypes.has(type)}
                  onChange={() => toggleType(type)}
                />
                <span className={type}>
                  {type.charAt(0).toUpperCase() + type.slice(1)}
                </span>
              </label>
            ),
          )}
        </div>
      )}

      <div className="diff-content">
        {isCrawlDiff ? (
          <CrawlDiffView
            diff={diffResult as CrawlDiffResult}
            visibleTypes={visibleTypes}
          />
        ) : (
          <ResearchDiffView diff={diffResult as ResearchDiffResult} />
        )}
      </div>
    </div>
  );
}

export default DiffViewer;
