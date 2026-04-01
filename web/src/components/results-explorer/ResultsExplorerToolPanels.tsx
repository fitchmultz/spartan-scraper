/**
 * Purpose: Render the optional analysis and assistant panels that branch off the primary results reader.
 * Responsibilities: Present tree, diff, visualization, transform, and AI assistant panels for the saved-results workspace.
 * Scope: Results workspace secondary tools only; route framing, result loading, and top-level export orchestration stay outside this file.
 * Usage: Import from `ResultsExplorer.tsx` or the local panel barrel when composing the `/jobs/:id` results surface.
 * Invariants/Assumptions: Secondary tools stay opt-in, the active tool config remains authoritative, and transform/shape actions run through parent-owned callbacks.
 */

import type { ComponentStatus, ExportShapeConfig } from "../../api";
import type { CrawlDiffResult, ResearchDiffResult } from "../../lib/diff-utils";
import type { TreeNode } from "../../lib/tree-utils";
import type { ClusterItem, EvidenceItem, Job, ResultItem } from "../../types";
import {
  ResultsAssistantSection,
  type ResultsAssistantMode,
} from "../ai-assistant";
import { ClusterGraph } from "../ClusterGraph";
import { DiffViewer } from "../DiffViewer";
import { EvidenceChart } from "../EvidenceChart";
import { TransformPreview } from "../TransformPreview";
import { TreeView } from "../TreeView";
import type {
  getAvailableSecondaryTools,
  SecondaryToolId,
  StatusFilter,
} from "./resultsExplorerUtils";

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
  resultEvidence: EvidenceItem[];
  resultClusters: ClusterItem[];
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
    format: "jsonl" | "json" | "md" | "csv" | "xlsx",
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
  selectedResult: ResultItem | null;
  mode: ResultsAssistantMode;
  onModeChange: (mode: ResultsAssistantMode) => void;
  shapeFormat: "md" | "csv" | "xlsx";
  onShapeFormatChange: (format: "md" | "csv" | "xlsx") => void;
  currentShape?: ExportShapeConfig;
  onApplyShape: (shape: ExportShapeConfig) => void;
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
