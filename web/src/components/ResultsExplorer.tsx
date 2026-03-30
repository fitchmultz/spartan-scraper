/**
 * Purpose: Provide the result-focused workspace for `/jobs/:id` with one dominant default reader.
 * Responsibilities: Compose the extracted results-workspace hook with the reader, promotion rail, secondary tools, and export surfaces.
 * Scope: Results exploration UI only; route framing and authoritative result fetching stay outside this component.
 * Usage: Render from `ResultsContainer` with the active job's saved results and the surrounding jobs list for compare workflows.
 * Invariants/Assumptions: A selected job ID exists before rendering, the default reader always remains visible on first paint, and comparison/export actions operate on saved job results.
 */

import type { ComponentStatus } from "../api";
import { isResearchResultItem } from "../lib/form-utils";
import type {
  AgenticResearchItem,
  CitationItem,
  ClusterItem,
  EvidenceItem,
  Job,
  ResultItem,
} from "../types";
import type { PromotionDestination } from "../types/promotion";
import { ResultsViewer } from "./ResultsViewer";
import { JobPromotionPanel } from "./promotion/JobPromotionPanel";
import {
  ExportOutcomeSummary,
  GuidedExportDrawer,
  ReaderToolbar,
  ResultsAssistantRail,
  ResultsToolPanel,
  SecondaryToolsDrawer,
} from "./results-explorer/ResultsExplorerPanels";
import { useResultsExplorer } from "./results-explorer/useResultsExplorer";

interface ResultsExplorerProps {
  jobId: string | null;
  resultItems: ResultItem[];
  selectedResultIndex: number;
  setSelectedResultIndex: (index: number) => void;
  resultSummary: string | null;
  resultConfidence: number | null;
  resultEvidence: EvidenceItem[];
  resultClusters: ClusterItem[];
  resultCitations: CitationItem[];
  resultAgentic: AgenticResearchItem | null;
  rawResult: string | null;
  resultFormat: string;
  currentPage: number;
  totalResults: number;
  resultsPerPage: number;
  onLoadPage: (page: number) => void;
  availableJobs: Job[];
  currentJob: Job | null;
  jobType?: "scrape" | "crawl" | "research";
  aiStatus?: ComponentStatus | null;
  onPromote: (
    destination: PromotionDestination,
    options?: {
      preferredExportFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx";
    },
  ) => void;
}

export function ResultsExplorer({
  jobId,
  resultItems,
  selectedResultIndex,
  setSelectedResultIndex,
  resultSummary,
  resultConfidence,
  resultEvidence,
  resultClusters,
  resultCitations,
  resultAgentic,
  rawResult,
  resultFormat,
  currentPage,
  totalResults,
  resultsPerPage,
  onLoadPage,
  availableJobs,
  currentJob,
  jobType = "crawl",
  aiStatus = null,
  onPromote,
}: ResultsExplorerProps) {
  const explorer = useResultsExplorer({
    jobId,
    resultItems,
    selectedResultIndex,
    setSelectedResultIndex,
    resultSummary,
    resultEvidence,
    currentJob,
    availableJobs,
    jobType,
    resultFormat,
    totalResults,
  });

  const activeResultItem = resultItems[explorer.activeResultIndex] ?? null;
  const activeResultSummary =
    activeResultItem && isResearchResultItem(activeResultItem)
      ? (activeResultItem.summary ?? null)
      : null;

  if (!jobId) {
    return null;
  }

  return (
    <div className="panel results-explorer">
      <div className="ai-assistant-surface">
        <div className="ai-assistant-surface__main">
          <div className="results-explorer-header">
            <div>
              <div className="results-viewer__section-label">Result reader</div>
              <h3>Read the saved output</h3>
              <p className="form-help">
                Start with the selected item, understand what changed, and only
                then branch into comparison, structure, transforms, or exports.
              </p>
              <div className="results-explorer__surface-summary">
                <span>Job {jobId}</span>
                {currentJob?.kind ? (
                  <span>{currentJob.kind} workflow</span>
                ) : null}
                <span>
                  {totalResults > 0
                    ? `${totalResults} saved results`
                    : "Saved result route"}
                </span>
              </div>
            </div>

            <div className="results-explorer__actions">
              <button
                type="button"
                className={explorer.isToolsOpen ? "active" : "secondary"}
                onClick={explorer.toggleToolsDrawer}
              >
                Tools
              </button>
              <button
                type="button"
                className={explorer.isExportOpen ? "active" : "secondary"}
                onClick={explorer.toggleExportDrawer}
              >
                Export
              </button>
              <button
                type="button"
                className="secondary"
                onClick={() => explorer.openResultsAssistant("shape")}
              >
                Open AI assistant
              </button>
            </div>
          </div>

          {currentJob?.status === "succeeded" ? (
            <JobPromotionPanel
              options={explorer.promotionOptions}
              onPromote={(destination) => onPromote(destination)}
            />
          ) : null}

          <ReaderToolbar
            searchQuery={explorer.searchQuery}
            statusFilter={explorer.statusFilter}
            visibleResults={explorer.filteredResultItems.length}
            totalResults={explorer.totalVisibleResults}
            onChangeSearchQuery={explorer.setSearchQuery}
            onChangeStatusFilter={explorer.setStatusFilter}
            onClearFilters={explorer.clearReaderFilters}
          />

          {explorer.filteredResultItems.length === 0 ? (
            <div className="results-explorer__surface-note">
              No saved results match the current reader filters.
            </div>
          ) : null}

          {explorer.isToolsOpen ? (
            <SecondaryToolsDrawer
              tools={explorer.secondaryTools}
              activeTool={explorer.activeTool}
              onSelectTool={explorer.setActiveTool}
              onClose={() => explorer.setIsToolsOpen(false)}
            />
          ) : null}

          {explorer.isExportOpen ? (
            <GuidedExportDrawer
              options={explorer.exportOptions}
              isExporting={explorer.isExporting}
              exportError={explorer.exportError}
              onExport={(format) => {
                void explorer.handleDirectExport(format);
              }}
              onClose={() => explorer.setIsExportOpen(false)}
              onOpenTransform={explorer.openTransformTool}
            />
          ) : null}

          {explorer.latestExportOutcome ? (
            <ExportOutcomeSummary
              outcome={explorer.latestExportOutcome}
              onPromoteExportSchedule={(preferredExportFormat) =>
                onPromote("export-schedule", { preferredExportFormat })
              }
            />
          ) : null}

          <ResultsViewer
            jobId={jobId}
            jobKind={
              currentJob?.kind as "scrape" | "crawl" | "research" | undefined
            }
            resultItems={explorer.filteredResultItems}
            selectedResultIndex={explorer.visibleSelectedIndex}
            setSelectedResultIndex={explorer.setSelectedVisibleResultIndex}
            resultSummary={resultSummary}
            resultConfidence={resultConfidence}
            resultEvidence={resultEvidence}
            resultClusters={resultClusters}
            resultCitations={resultCitations}
            resultAgentic={resultAgentic}
            rawResult={rawResult}
            resultFormat={resultFormat}
            currentPage={currentPage}
            totalResults={totalResults}
            resultsPerPage={resultsPerPage}
            onLoadPage={onLoadPage}
            onOpenResearchAssistant={() =>
              explorer.openResultsAssistant("research")
            }
          />

          <ResultsToolPanel
            activeTool={explorer.activeTool}
            activeToolConfig={explorer.activeToolConfig}
            treeNodes={explorer.treeNodes}
            treeSelectedId={explorer.treeSelectedId}
            treeExpandedIds={explorer.treeExpandedIds}
            searchQuery={explorer.searchQuery}
            statusFilter={explorer.statusFilter}
            compareJobId={explorer.compareJobId}
            comparableJobs={explorer.comparableJobs}
            currentJob={currentJob}
            compareJob={explorer.compareJob}
            diffResult={explorer.diffResult}
            diffLoading={explorer.diffLoading}
            diffError={explorer.diffError}
            isResearchJob={explorer.isResearchJob}
            resultEvidence={resultEvidence}
            resultClusters={resultClusters}
            selectedEvidenceUrl={explorer.selectedEvidenceUrl}
            selectedClusterId={explorer.selectedClusterId}
            exportError={explorer.exportError}
            jobId={jobId}
            aiStatus={aiStatus}
            shapeExportFormat={explorer.shapeExportFormat}
            shapeConfigText={explorer.shapeConfigText}
            shapeConfigError={explorer.shapeConfigError}
            onCloseTool={() => explorer.setActiveTool(null)}
            onExpandAllTreeNodes={explorer.expandAllTreeNodes}
            onCollapseAllTreeNodes={explorer.collapseAllTreeNodes}
            onTreeSelect={explorer.handleTreeSelect}
            onTreeToggle={explorer.handleTreeToggle}
            onChangeCompareJobID={explorer.setCompareJobId}
            onSelectEvidenceUrl={explorer.setSelectedEvidenceUrl}
            onSelectClusterId={explorer.setSelectedClusterId}
            onTransformApply={(format, expression, language) => {
              void explorer.handleExportWithTransform(
                format,
                expression,
                language,
              );
            }}
            onShapeFormatChange={explorer.setShapeExportFormat}
            onOpenShapeAssistant={() => explorer.openResultsAssistant("shape")}
            onClearShape={explorer.clearShapeConfig}
            onShapeConfigTextChange={explorer.updateShapeConfigText}
            onShapeExport={() => {
              void explorer.handleShapeExport();
            }}
          />
        </div>

        <ResultsAssistantRail
          jobId={jobId}
          jobType={jobType}
          resultFormat={resultFormat}
          aiStatus={aiStatus}
          selectedResultIndex={explorer.activeResultIndex}
          resultSummary={activeResultSummary}
          selectedResult={activeResultItem}
          mode={explorer.assistantMode}
          onModeChange={explorer.setAssistantMode}
          shapeFormat={explorer.shapeExportFormat}
          onShapeFormatChange={explorer.setShapeExportFormat}
          currentShape={explorer.currentShapeConfig}
          onApplyShape={explorer.applyAssistantShape}
        />
      </div>
    </div>
  );
}

export default ResultsExplorer;
