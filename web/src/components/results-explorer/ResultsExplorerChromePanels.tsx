/**
 * Purpose: Render the compact results-workspace chrome that surrounds the primary reader.
 * Responsibilities: Present quick actions, reader filters, secondary-tool entry points, guided export controls, and export outcome summaries.
 * Scope: Results workspace presentation only; result selection, export side effects, and route orchestration stay outside this file.
 * Usage: Import from `ResultsExplorer.tsx` or the local panel barrel to compose the `/jobs/:id` workspace.
 * Invariants/Assumptions: The reader remains primary, quick actions stay explicit, and export actions only run through parent-owned callbacks.
 */

import type { ExportInspection } from "../../api";
import type { PromotionOption } from "../../types/promotion";
import {
  getPrimaryExportGuidanceOptions,
  type ExportFormat,
  type StatusFilter,
} from "./resultsExplorerUtils";
import type {
  getAvailableSecondaryTools,
  getExportGuidanceOptions,
} from "./resultsExplorerUtils";

interface ResultsQuickActionRailProps {
  promotionOptions: PromotionOption[];
  exportOptions: ReturnType<typeof getExportGuidanceOptions>;
  isExporting: boolean;
  onPromote: (option: PromotionOption) => void;
  onExport: (format: ExportFormat) => void;
  onOpenExport: () => void;
}

interface ReaderToolbarProps {
  searchQuery: string;
  statusFilter: StatusFilter;
  visibleResults: number;
  totalResults: number;
  onChangeSearchQuery: (value: string) => void;
  onChangeStatusFilter: (value: StatusFilter) => void;
  onClearFilters: () => void;
}

interface SecondaryToolsDrawerProps {
  tools: ReturnType<typeof getAvailableSecondaryTools>;
  activeTool:
    | ReturnType<typeof getAvailableSecondaryTools>[number]["id"]
    | null;
  onSelectTool: (
    tool: ReturnType<typeof getAvailableSecondaryTools>[number]["id"],
  ) => void;
  onClose: () => void;
}

interface GuidedExportDrawerProps {
  options: ReturnType<typeof getExportGuidanceOptions>;
  isExporting: boolean;
  exportError: string | null;
  onExport: (format: ExportFormat) => void;
  onClose: () => void;
  onOpenTransform: () => void;
}

export function ResultsQuickActionRail({
  promotionOptions,
  exportOptions,
  isExporting,
  onPromote,
  onExport,
  onOpenExport,
}: ResultsQuickActionRailProps) {
  const directExportOptions = getPrimaryExportGuidanceOptions(exportOptions);
  const eligiblePromotionOptions = promotionOptions.filter(
    (option) => option.eligible,
  );

  return (
    <section
      className="results-explorer__quick-actions"
      aria-label="Job quick actions"
    >
      <div className="results-explorer__quick-actions-copy">
        <div className="results-viewer__section-label">Operator actions</div>
        <h4>Inspect, export, or promote without leaving the fold</h4>
        <p className="form-help">
          Keep the common next steps one tap away, then open the deeper
          promotion and export guidance only when you need the full context.
        </p>
      </div>

      <div className="results-explorer__quick-actions-groups">
        {eligiblePromotionOptions.length > 0 ? (
          <div className="results-explorer__quick-actions-group">
            <strong>Promote this job</strong>
            <div className="results-explorer__quick-actions-buttons">
              {eligiblePromotionOptions.map((option) => (
                <button
                  key={option.destination}
                  type="button"
                  className="secondary"
                  onClick={() => onPromote(option)}
                >
                  {option.title}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        <div className="results-explorer__quick-actions-group">
          <strong>Export the saved output</strong>
          <div className="results-explorer__quick-actions-buttons">
            {directExportOptions.map((option) => (
              <button
                key={`top-${option.format}`}
                type="button"
                onClick={() => onExport(option.format)}
                disabled={isExporting}
              >
                Export {option.title}
              </button>
            ))}
            <button type="button" className="secondary" onClick={onOpenExport}>
              More export options
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}

export function ReaderToolbar({
  searchQuery,
  statusFilter,
  visibleResults,
  totalResults,
  onChangeSearchQuery,
  onChangeStatusFilter,
  onClearFilters,
}: ReaderToolbarProps) {
  return (
    <div className="results-explorer-toolbar">
      <div className="search-box">
        <input
          type="text"
          placeholder="Search by URL, title, or content..."
          value={searchQuery}
          onChange={(event) => onChangeSearchQuery(event.target.value)}
        />
        {searchQuery ? (
          <button
            type="button"
            className="search-clear"
            onClick={() => onChangeSearchQuery("")}
            aria-label="Clear search"
          >
            ×
          </button>
        ) : null}
      </div>

      <select
        value={statusFilter}
        onChange={(event) =>
          onChangeStatusFilter(event.target.value as StatusFilter)
        }
        className="status-filter"
        aria-label="Result status filter"
      >
        <option value="all">All status</option>
        <option value="success">Success (2xx)</option>
        <option value="error">Error (4xx/5xx)</option>
      </select>

      <div className="results-explorer__toolbar-hint">
        {visibleResults === totalResults || totalResults === 0
          ? `${Math.max(visibleResults, totalResults)} result${
              Math.max(visibleResults, totalResults) === 1 ? "" : "s"
            } in the reader.`
          : `${visibleResults} of ${totalResults} results are visible in the reader.`}
      </div>

      {(searchQuery.trim() || statusFilter !== "all") &&
      visibleResults !== totalResults ? (
        <button type="button" className="secondary" onClick={onClearFilters}>
          Clear reader filters
        </button>
      ) : null}
    </div>
  );
}

export function SecondaryToolsDrawer({
  tools,
  activeTool,
  onSelectTool,
  onClose,
}: SecondaryToolsDrawerProps) {
  return (
    <div className="results-explorer__drawer">
      <div className="results-explorer__drawer-header">
        <div>
          <div className="results-viewer__section-label">Secondary tools</div>
          <h4>Open comparison and analysis only when you need it</h4>
          <p className="form-help">
            The reader stays primary. These tools sit underneath it so you can
            branch into structure, comparison, visualization, or transforms
            without losing context.
          </p>
        </div>
        <button type="button" className="secondary" onClick={onClose}>
          Close
        </button>
      </div>

      <div className="results-explorer__tool-grid">
        {tools.map((tool) => (
          <button
            key={tool.id}
            type="button"
            className={`results-explorer__tool-card ${
              activeTool === tool.id ? "is-active" : ""
            }`}
            onClick={() => onSelectTool(tool.id)}
          >
            <strong>{tool.label}</strong>
            <span>{tool.description}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

export function GuidedExportDrawer({
  options,
  isExporting,
  exportError,
  onExport,
  onClose,
  onOpenTransform,
}: GuidedExportDrawerProps) {
  const scopePreview = options[0];
  const primaryOptions = getPrimaryExportGuidanceOptions(options);

  return (
    <div className="results-explorer__drawer">
      <div className="results-explorer__drawer-header">
        <div>
          <div className="results-viewer__section-label">Guided export</div>
          <h4>Choose the right handoff format before you download</h4>
          <p className="form-help">
            Export stays quiet by default. Open it when you are ready to hand
            off or archive the saved result.
          </p>
        </div>
        <div className="results-explorer__export-actions">
          <button type="button" className="secondary" onClick={onOpenTransform}>
            Need a transformed export?
          </button>
          <button type="button" className="secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>

      {scopePreview ? (
        <div className="results-explorer__export-preview">
          <strong>{scopePreview.scopeLabel}</strong>
          <p>{scopePreview.scopeNote}</p>
        </div>
      ) : null}

      {primaryOptions.length > 0 ? (
        <div className="results-explorer__export-quick-start">
          <div className="results-explorer__export-quick-copy">
            <strong>Start with the direct handoff</strong>
            <p>
              Keep the most common downloads visible while you scan the rest of
              the export guidance.
            </p>
          </div>
          <div className="results-explorer__export-quick-actions">
            {primaryOptions.map((option) => (
              <button
                key={`quick-${option.format}`}
                type="button"
                onClick={() => onExport(option.format)}
                disabled={isExporting}
              >
                Export {option.title} now
              </button>
            ))}
          </div>
        </div>
      ) : null}

      {exportError ? (
        <div className="transform-error">{exportError}</div>
      ) : null}

      <div className="results-explorer__export-grid">
        {options.map((option) => (
          <div key={option.format} className="results-explorer__export-card">
            <div className="results-explorer__export-card-head">
              <div>
                <h5>{option.title}</h5>
                <p>{option.description}</p>
              </div>
              <span
                className={`results-explorer__readiness results-explorer__readiness--${option.readiness}`}
              >
                {option.readiness}
              </span>
            </div>
            <button
              type="button"
              className="secondary"
              onClick={() => onExport(option.format)}
              disabled={isExporting}
            >
              Export {option.title}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

export function ExportOutcomeSummary({
  outcome,
  onPromoteExportSchedule,
}: {
  outcome: ExportInspection;
  onPromoteExportSchedule?: (
    format?: "json" | "jsonl" | "md" | "csv" | "xlsx",
  ) => void;
}) {
  return (
    <div
      className="panel"
      style={{
        marginBottom: 16,
        border: "1px solid var(--stroke)",
        background: "rgba(255, 255, 255, 0.02)",
      }}
    >
      <div className="results-viewer__section-label">Latest export outcome</div>
      <div
        className="row"
        style={{ justifyContent: "space-between", alignItems: "flex-start" }}
      >
        <div>
          <h4 style={{ margin: 0 }}>{outcome.title}</h4>
          <p className="form-help" style={{ marginTop: 8 }}>
            {outcome.message}
          </p>
        </div>
        <strong>{outcome.status}</strong>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
          gap: 12,
          marginTop: 16,
        }}
      >
        <div>
          <strong>Export ID</strong>
          <div style={{ fontFamily: "monospace", fontSize: 12 }}>
            {outcome.id}
          </div>
        </div>
        <div>
          <strong>Requested format</strong>
          <div>{outcome.request.format}</div>
        </div>
        <div>
          <strong>Destination</strong>
          <div style={{ wordBreak: "break-word" }}>
            {outcome.destination || "-"}
          </div>
        </div>
        <div>
          <strong>Artifact</strong>
          <div>{outcome.artifact?.filename || "Not available"}</div>
        </div>
      </div>

      {outcome.failure ? (
        <div className="transform-error" style={{ marginTop: 16 }}>
          {outcome.failure.category}: {outcome.failure.summary}
        </div>
      ) : null}

      {outcome.actions?.length ? (
        <div style={{ marginTop: 16 }}>
          <strong>Suggested next steps</strong>
          <ul style={{ margin: "8px 0 0", paddingLeft: 20 }}>
            {outcome.actions.map((action) => (
              <li key={`${action.kind}-${action.label}-${action.value}`}>
                <strong>{action.label}</strong>
                {action.value ? ` — ${action.value}` : ""}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {onPromoteExportSchedule ? (
        <div style={{ marginTop: 16 }}>
          <button
            type="button"
            className="secondary"
            onClick={() =>
              onPromoteExportSchedule(
                outcome.request.format as
                  | "json"
                  | "jsonl"
                  | "md"
                  | "csv"
                  | "xlsx"
                  | undefined,
              )
            }
          >
            Create recurring export from this result
          </button>
        </div>
      ) : null}
    </div>
  );
}
