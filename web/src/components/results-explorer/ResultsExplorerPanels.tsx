/**
 * Purpose: Break the saved-results workspace chrome into dedicated toolbar, tool, export, and assistant panels.
 * Responsibilities: Render reader filters, secondary-tool drawers, export guidance, tool-specific panels, and the results AI assistant rail without forcing `ResultsExplorer.tsx` to inline every branch.
 * Scope: Results workspace presentation only; selection state, diff loading, export actions, and route-level orchestration stay in `ResultsExplorer.tsx`.
 * Usage: Imported by `ResultsExplorer.tsx` to compose the `/jobs/:id` reader surface.
 * Invariants/Assumptions: The default reader stays primary, secondary tools remain explicit opt-ins, and assistant actions only apply through explicit callbacks.
 */

import type {
  ComponentStatus,
  ExportInspection,
  ExportShapeConfig,
} from "../../api";
import type { CrawlDiffResult, ResearchDiffResult } from "../../lib/diff-utils";
import type { TreeNode } from "../../lib/tree-utils";
import type { Job } from "../../types";
import {
  ResultsAssistantSection,
  type ResultsAssistantMode,
} from "../ai-assistant";
import { ClusterGraph } from "../ClusterGraph";
import { DiffViewer } from "../DiffViewer";
import { EvidenceChart } from "../EvidenceChart";
import type {
  ExportFormat,
  SecondaryToolId,
  StatusFilter,
} from "./resultsExplorerUtils";
import type {
  getAvailableSecondaryTools,
  getExportGuidanceOptions,
} from "./resultsExplorerUtils";
import { TransformPreview } from "../TransformPreview";
import { TreeView } from "../TreeView";

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
  activeTool: SecondaryToolId | null;
  onSelectTool: (tool: SecondaryToolId) => void;
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

interface ResultsToolPanelProps {
  activeTool: SecondaryToolId | null;
  activeToolConfig:
    | ReturnType<typeof getAvailableSecondaryTools>[number]
    | undefined;
  treeNodes: TreeNode[];
  treeSelectedId: string | null;
  treeExpandedIds: Set<string>;
  searchQuery: string;
  statusFilter: StatusFilter;
  compareJobId: string | null;
  comparableJobs: Job[];
  currentJob: Job | null;
  compareJob: Job | null;
  diffResult: CrawlDiffResult | ResearchDiffResult | null;
  diffLoading: boolean;
  diffError: string | null;
  isResearchJob: boolean;
  resultEvidence: import("../../types").EvidenceItem[];
  resultClusters: import("../../types").ClusterItem[];
  selectedEvidenceUrl: string | null;
  selectedClusterId: string | null;
  exportError: string | null;
  jobId: string;
  aiStatus?: ComponentStatus | null;
  shapeExportFormat: "md" | "csv" | "xlsx";
  shapeConfigText: string;
  shapeConfigError: string | null;
  onCloseTool: () => void;
  onExpandAllTreeNodes: () => void;
  onCollapseAllTreeNodes: () => void;
  onTreeSelect: (node: TreeNode) => void;
  onTreeToggle: (nodeId: string) => void;
  onChangeCompareJobID: (jobID: string | null) => void;
  onSelectEvidenceUrl: (url: string) => void;
  onSelectClusterId: (clusterId: string) => void;
  onTransformApply: (
    format: ExportFormat,
    expression: string,
    language: "jmespath" | "jsonata",
  ) => void;
  onShapeFormatChange: (format: "md" | "csv" | "xlsx") => void;
  onOpenShapeAssistant: () => void;
  onClearShape: () => void;
  onShapeConfigTextChange: (value: string) => void;
  onShapeExport: () => void;
}

interface ResultsAssistantRailProps {
  jobId: string;
  jobType: "scrape" | "crawl" | "research";
  resultFormat: string;
  aiStatus?: ComponentStatus | null;
  selectedResultIndex: number;
  resultSummary: string | null;
  selectedResult: import("../../types").ResultItem | null;
  mode: ResultsAssistantMode;
  onModeChange: (mode: ResultsAssistantMode) => void;
  shapeFormat: "md" | "csv" | "xlsx";
  onShapeFormatChange: (format: "md" | "csv" | "xlsx") => void;
  currentShape?: ExportShapeConfig;
  onApplyShape: (shape: ExportShapeConfig) => void;
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
  const recommendedOptions = options.filter(
    (option) => option.readiness === "recommended",
  );
  const primaryOptions = (
    recommendedOptions.length > 0 ? recommendedOptions : options
  ).slice(0, 2);

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

function ExplorerTreeControls({
  treeNodes,
  onExpandAll,
  onCollapseAll,
}: {
  treeNodes: TreeNode[];
  onExpandAll: () => void;
  onCollapseAll: () => void;
}) {
  return (
    <div className="tree-controls">
      <button type="button" className="secondary" onClick={onExpandAll}>
        Expand all
      </button>
      <button type="button" className="secondary" onClick={onCollapseAll}>
        Collapse to domains
      </button>
      <span className="tree-stats">
        {treeNodes.length} domains,{" "}
        {treeNodes.reduce((sum, node) => sum + node.resultCount, 0)} pages
      </span>
    </div>
  );
}

function ExplorerDiffControls({
  compareJobId,
  comparableJobs,
  onChangeCompareJobID,
}: {
  compareJobId: string | null;
  comparableJobs: Job[];
  onChangeCompareJobID: (jobID: string | null) => void;
}) {
  return (
    <div className="diff-controls">
      <label>
        Compare with:
        <select
          value={compareJobId || ""}
          onChange={(event) => onChangeCompareJobID(event.target.value || null)}
        >
          <option value="">Select a job...</option>
          {comparableJobs.map((job) => (
            <option key={job.id} value={job.id}>
              {job.id} ({job.status})
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}

export function ResultsToolPanel({
  activeTool,
  activeToolConfig,
  treeNodes,
  treeSelectedId,
  treeExpandedIds,
  searchQuery,
  statusFilter,
  compareJobId,
  comparableJobs,
  currentJob,
  compareJob,
  diffResult,
  diffLoading,
  diffError,
  isResearchJob,
  resultEvidence,
  resultClusters,
  selectedEvidenceUrl,
  selectedClusterId,
  exportError,
  jobId,
  aiStatus = null,
  shapeExportFormat,
  shapeConfigText,
  shapeConfigError,
  onCloseTool,
  onExpandAllTreeNodes,
  onCollapseAllTreeNodes,
  onTreeSelect,
  onTreeToggle,
  onChangeCompareJobID,
  onSelectEvidenceUrl,
  onSelectClusterId,
  onTransformApply,
  onShapeFormatChange,
  onOpenShapeAssistant,
  onClearShape,
  onShapeConfigTextChange,
  onShapeExport,
}: ResultsToolPanelProps) {
  if (!activeToolConfig) {
    return null;
  }

  return (
    <div className="results-explorer__tool-panel">
      <div className="results-explorer__drawer-header">
        <div>
          <div className="results-viewer__section-label">Secondary tool</div>
          <h4>Secondary tool: {activeToolConfig.label}</h4>
          <p className="form-help">{activeToolConfig.description}</p>
        </div>
        <button type="button" className="secondary" onClick={onCloseTool}>
          Hide tool
        </button>
      </div>

      {activeTool === "tree" ? (
        <>
          <ExplorerTreeControls
            treeNodes={treeNodes}
            onExpandAll={onExpandAllTreeNodes}
            onCollapseAll={onCollapseAllTreeNodes}
          />
          <TreeView
            nodes={treeNodes}
            selectedId={treeSelectedId}
            onSelect={onTreeSelect}
            onToggleExpand={onTreeToggle}
            expandedIds={treeExpandedIds}
            searchQuery={searchQuery}
            statusFilter={statusFilter}
          />
        </>
      ) : null}

      {activeTool === "diff" ? (
        <>
          <ExplorerDiffControls
            compareJobId={compareJobId}
            comparableJobs={comparableJobs}
            onChangeCompareJobID={onChangeCompareJobID}
          />
          <DiffViewer
            baseJob={currentJob}
            compareJob={compareJob}
            diffResult={diffResult}
            isLoading={diffLoading}
            error={diffError}
            onClose={onCloseTool}
          />
        </>
      ) : null}

      {activeTool === "visualize" && isResearchJob ? (
        <div className="visualize-content">
          <EvidenceChart
            evidence={resultEvidence}
            clusters={resultClusters}
            selectedEvidenceUrl={selectedEvidenceUrl}
            onSelectEvidence={(item) => onSelectEvidenceUrl(item.url)}
          />
          {resultClusters.length > 0 ? (
            <ClusterGraph
              clusters={resultClusters}
              evidence={resultEvidence}
              selectedClusterId={selectedClusterId}
              onSelectCluster={(cluster) => onSelectClusterId(cluster.id)}
            />
          ) : null}
        </div>
      ) : null}

      {activeTool === "transform" ? (
        <div className="results-explorer__transform-stack">
          {exportError ? (
            <div className="transform-error">{exportError}</div>
          ) : null}
          <TransformPreview
            jobId={jobId}
            aiStatus={aiStatus}
            onApply={onTransformApply}
          />
          <div className="panel results-explorer__shape-export">
            <h4>Direct shape export</h4>
            <p className="form-help">
              Apply a bounded export shape directly to this saved result for
              markdown and tabular handoffs.
            </p>
            <div className="row results-explorer__shape-export-controls">
              <label>
                Format
                <select
                  value={shapeExportFormat}
                  onChange={(event) =>
                    onShapeFormatChange(
                      event.target.value as "md" | "csv" | "xlsx",
                    )
                  }
                >
                  <option value="md">Markdown</option>
                  <option value="csv">CSV</option>
                  <option value="xlsx">XLSX</option>
                </select>
              </label>
              <button
                type="button"
                className="secondary"
                onClick={onOpenShapeAssistant}
              >
                Open AI assistant
              </button>
              <button
                type="button"
                className="secondary"
                onClick={onClearShape}
              >
                Clear shape
              </button>
              <button type="button" onClick={onShapeExport}>
                Export shaped result
              </button>
            </div>
            <textarea
              className="form-textarea results-explorer__shape-export-input"
              rows={10}
              value={shapeConfigText}
              onChange={(event) => onShapeConfigTextChange(event.target.value)}
              placeholder='{"summaryFields":["title","url"],"normalizedFields":["field.price"]}'
            />
            {shapeConfigError ? (
              <div className="transform-error">Error: {shapeConfigError}</div>
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function ResultsAssistantRail({
  jobId,
  jobType,
  resultFormat,
  aiStatus = null,
  selectedResultIndex,
  resultSummary,
  selectedResult,
  mode,
  onModeChange,
  shapeFormat,
  onShapeFormatChange,
  currentShape,
  onApplyShape,
}: ResultsAssistantRailProps) {
  return (
    <ResultsAssistantSection
      key={`${jobId}:${selectedResultIndex}`}
      jobId={jobId}
      jobType={jobType}
      resultFormat={resultFormat}
      aiStatus={aiStatus}
      selectedResultIndex={selectedResultIndex}
      resultSummary={resultSummary}
      selectedResult={selectedResult}
      mode={mode}
      onModeChange={onModeChange}
      shapeFormat={shapeFormat}
      onShapeFormatChange={onShapeFormatChange}
      currentShape={currentShape}
      onApplyShape={onApplyShape}
    />
  );
}
