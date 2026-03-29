/**
 * Purpose: Own reader filter state, tree navigation state, and filtered-item selection mapping for the results workspace.
 * Responsibilities: Manage search query, status filter, URL tree expansion/selection, evidence/cluster selection, and the mapping between filtered and source indexes.
 * Scope: Read-only selection and filtering only; no export, compare, or assistant logic.
 * Usage: Called from `useResultsExplorer` (or directly when only selection state is needed).
 * Invariants/Assumptions: filteredSourceIndexes stays aligned with filteredResultItems; tree state is only meaningful for crawl-type results.
 */

import { useEffect, useMemo, useState } from "react";

import { buildUrlTree, type TreeNode } from "../../lib/tree-utils";
import type { CrawlResultItem, EvidenceItem, ResultItem } from "../../types";
import {
  buildDefaultExpandedTreeIds,
  collectTreeNodeIds,
  filterResultItems,
  hasResearchVisualization,
  isCrawlResult,
  type StatusFilter,
} from "./resultsExplorerUtils";

export interface UseResultsSelectionStateOptions {
  resultItems: ResultItem[];
  selectedResultIndex: number;
  setSelectedResultIndex: (index: number) => void;
  resultEvidence: EvidenceItem[];
  jobType: "scrape" | "crawl" | "research";
  totalResults: number;
}

export function useResultsSelectionState({
  resultItems,
  selectedResultIndex,
  setSelectedResultIndex,
  resultEvidence,
  jobType,
  totalResults,
}: UseResultsSelectionStateOptions) {
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [treeExpandedIds, setTreeExpandedIds] = useState<Set<string>>(
    new Set(),
  );
  const [treeSelectedId, setTreeSelectedId] = useState<string | null>(null);
  const [selectedEvidenceUrl, setSelectedEvidenceUrl] = useState<string | null>(
    null,
  );
  const [selectedClusterId, setSelectedClusterId] = useState<string | null>(
    null,
  );

  const isResearchJob = hasResearchVisualization(jobType, resultEvidence);

  const treeNodes = useMemo(() => {
    const crawlItems = resultItems.filter((item): item is CrawlResultItem =>
      isCrawlResult(item),
    );
    return buildUrlTree(crawlItems);
  }, [resultItems]);

  useEffect(() => {
    if (treeNodes.length > 0 && treeExpandedIds.size === 0) {
      setTreeExpandedIds(buildDefaultExpandedTreeIds(treeNodes));
    }
  }, [treeNodes, treeExpandedIds.size]);

  const filteredResultItems = useMemo(
    () => filterResultItems(resultItems, searchQuery, statusFilter),
    [resultItems, searchQuery, statusFilter],
  );

  const filteredSourceIndexes = useMemo(
    () =>
      resultItems.reduce<number[]>((indexes, item, index) => {
        if (filteredResultItems.includes(item)) {
          indexes.push(index);
        }
        return indexes;
      }, []),
    [filteredResultItems, resultItems],
  );

  useEffect(() => {
    if (filteredSourceIndexes.length === 0) {
      return;
    }

    if (!filteredSourceIndexes.includes(selectedResultIndex)) {
      setSelectedResultIndex(filteredSourceIndexes[0]);
    }
  }, [filteredSourceIndexes, selectedResultIndex, setSelectedResultIndex]);

  const visibleSelectedIndex = Math.max(
    0,
    filteredSourceIndexes.indexOf(selectedResultIndex),
  );

  const setSelectedVisibleResultIndex = (visibleIndex: number) => {
    const sourceIndex = filteredSourceIndexes[visibleIndex];
    if (typeof sourceIndex === "number") {
      setSelectedResultIndex(sourceIndex);
    }
  };

  const handleTreeSelect = (node: TreeNode) => {
    setTreeSelectedId(node.id);
    if (!node.result) {
      return;
    }

    const index = resultItems.findIndex(
      (item) => isCrawlResult(item) && item.url === node.url,
    );
    if (index !== -1) {
      setSelectedResultIndex(index);
    }
  };

  const handleTreeToggle = (nodeId: string) => {
    setTreeExpandedIds((previous) => {
      const next = new Set(previous);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  };

  const expandAllTreeNodes = () => {
    setTreeExpandedIds(collectTreeNodeIds(treeNodes));
  };

  const collapseAllTreeNodes = () => {
    setTreeExpandedIds(buildDefaultExpandedTreeIds(treeNodes));
  };

  const clearReaderFilters = () => {
    setSearchQuery("");
    setStatusFilter("all");
  };

  return {
    clearReaderFilters,
    collapseAllTreeNodes,
    expandAllTreeNodes,
    filteredResultItems,
    filteredSourceIndexes,
    handleTreeSelect,
    handleTreeToggle,
    isResearchJob,
    searchQuery,
    selectedClusterId,
    selectedEvidenceUrl,
    setSearchQuery,
    setSelectedClusterId,
    setSelectedEvidenceUrl,
    setSelectedVisibleResultIndex,
    setStatusFilter,
    statusFilter,
    totalVisibleResults: Math.max(totalResults, resultItems.length),
    treeExpandedIds,
    treeNodes,
    treeSelectedId,
    visibleSelectedIndex,
  };
}
