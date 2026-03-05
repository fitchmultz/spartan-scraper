/**
 * Diff Utilities Module
 *
 * Provides functions for comparing crawl and research results between job runs.
 * Computes added, removed, modified, and unchanged items with detailed field
 * change tracking for visualization in diff views.
 *
 * @module diff-utils
 */
import type {
  CrawlResultItem,
  ResearchResultItem,
  ResultItem,
  EvidenceItem,
  ClusterItem,
  CitationItem,
} from "../types";

/**
 * Represents a change to a specific field.
 */
export interface FieldChange {
  /** Field name that changed */
  field: string;
  /** Previous value */
  oldValue: unknown;
  /** New value */
  newValue: unknown;
}

/**
 * Result of comparing two sets of crawl results.
 */
export interface CrawlDiffResult {
  /** Items present in 'after' but not in 'before' */
  added: CrawlResultItem[];
  /** Items present in 'before' but not in 'after' */
  removed: CrawlResultItem[];
  /** Items present in both with field-level changes */
  modified: {
    url: string;
    before: CrawlResultItem;
    after: CrawlResultItem;
    changes: FieldChange[];
  }[];
  /** Items that are identical in both sets */
  unchanged: CrawlResultItem[];
}

/**
 * Result of comparing two research results.
 */
export interface ResearchDiffResult {
  /** Summary changes */
  summaryChanges?: {
    oldValue: string | undefined;
    newValue: string | undefined;
  };
  /** Confidence score changes */
  confidenceChanges?: {
    oldValue: number | undefined;
    newValue: number | undefined;
  };
  /** Evidence items added */
  evidenceAdded: EvidenceItem[];
  /** Evidence items removed */
  evidenceRemoved: EvidenceItem[];
  /** Evidence items modified (by URL match) */
  evidenceModified: {
    url: string;
    before: EvidenceItem;
    after: EvidenceItem;
    changes: FieldChange[];
  }[];
  /** Evidence unchanged */
  evidenceUnchanged: EvidenceItem[];
  /** Clusters added */
  clustersAdded: ClusterItem[];
  /** Clusters removed */
  clustersRemoved: ClusterItem[];
  /** Clusters modified (by ID match) */
  clustersModified: {
    id: string;
    before: ClusterItem;
    after: ClusterItem;
    changes: FieldChange[];
  }[];
  /** Citations added */
  citationsAdded: CitationItem[];
  /** Citations removed */
  citationsRemoved: CitationItem[];
}

/**
 * Compute a unique key for a crawl result item.
 * Uses the URL as the primary identifier.
 *
 * @param result - Crawl result item
 * @returns Unique key string
 */
export function computeCrawlResultKey(result: CrawlResultItem): string {
  return result.url;
}

/**
 * Compute a unique key for an evidence item.
 *
 * @param evidence - Evidence item
 * @returns Unique key string
 */
export function computeEvidenceKey(evidence: EvidenceItem): string {
  return evidence.url;
}

/**
 * Compute a unique key for a cluster item.
 *
 * @param cluster - Cluster item
 * @returns Unique key string
 */
export function computeClusterKey(cluster: ClusterItem): string {
  return cluster.id;
}

/**
 * Compute a unique key for a citation item.
 *
 * @param citation - Citation item
 * @returns Unique key string
 */
export function computeCitationKey(citation: CitationItem): string {
  return citation.canonical || citation.url || "";
}

/**
 * Compare two values for equality.
 * Handles primitives, arrays, and objects.
 *
 * @param a - First value
 * @param b - Second value
 * @returns True if values are equal
 */
function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (typeof a !== typeof b) return false;
  if (a == null || b == null) return a === b;

  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false;
    return a.every((item, index) => deepEqual(item, b[index]));
  }

  if (typeof a === "object" && typeof b === "object") {
    const aObj = a as Record<string, unknown>;
    const bObj = b as Record<string, unknown>;
    const aKeys = Object.keys(aObj);
    const bKeys = Object.keys(bObj);
    if (aKeys.length !== bKeys.length) return false;
    return aKeys.every((key) => deepEqual(aObj[key], bObj[key]));
  }

  return false;
}

/**
 * Compute field-level changes between two crawl result items.
 *
 * @param before - Previous crawl result
 * @param after - Current crawl result
 * @returns Array of field changes
 */
export function computeCrawlFieldChanges(
  before: CrawlResultItem,
  after: CrawlResultItem,
): FieldChange[] {
  const changes: FieldChange[] = [];
  const fields: (keyof CrawlResultItem)[] = [
    "status",
    "title",
    "text",
    "links",
    "metadata",
    "extracted",
    "normalized",
  ];

  for (const field of fields) {
    const oldValue = before[field];
    const newValue = after[field];

    if (!deepEqual(oldValue, newValue)) {
      changes.push({ field, oldValue, newValue });
    }
  }

  return changes;
}

/**
 * Compute field-level changes between two evidence items.
 *
 * @param before - Previous evidence
 * @param after - Current evidence
 * @returns Array of field changes
 */
export function computeEvidenceFieldChanges(
  before: EvidenceItem,
  after: EvidenceItem,
): FieldChange[] {
  const changes: FieldChange[] = [];
  const fields: (keyof EvidenceItem)[] = [
    "title",
    "snippet",
    "score",
    "confidence",
    "citationUrl",
    "clusterId",
  ];

  for (const field of fields) {
    const oldValue = before[field];
    const newValue = after[field];

    if (!deepEqual(oldValue, newValue)) {
      changes.push({ field, oldValue, newValue });
    }
  }

  return changes;
}

/**
 * Compute field-level changes between two cluster items.
 *
 * @param before - Previous cluster
 * @param after - Current cluster
 * @returns Array of field changes
 */
export function computeClusterFieldChanges(
  before: ClusterItem,
  after: ClusterItem,
): FieldChange[] {
  const changes: FieldChange[] = [];
  const fields: (keyof ClusterItem)[] = ["label", "confidence", "evidence"];

  for (const field of fields) {
    const oldValue = before[field];
    const newValue = after[field];

    if (!deepEqual(oldValue, newValue)) {
      changes.push({ field, oldValue, newValue });
    }
  }

  return changes;
}

/**
 * Compare two sets of crawl results.
 *
 * @param before - Previous crawl results
 * @param after - Current crawl results
 * @returns Detailed diff result
 */
export function diffCrawlResults(
  before: CrawlResultItem[],
  after: CrawlResultItem[],
): CrawlDiffResult {
  const beforeMap = new Map<string, CrawlResultItem>();
  const afterMap = new Map<string, CrawlResultItem>();

  for (const item of before) {
    beforeMap.set(computeCrawlResultKey(item), item);
  }

  for (const item of after) {
    afterMap.set(computeCrawlResultKey(item), item);
  }

  const added: CrawlResultItem[] = [];
  const removed: CrawlResultItem[] = [];
  const modified: CrawlDiffResult["modified"] = [];
  const unchanged: CrawlResultItem[] = [];

  // Find added and modified items
  for (const [key, afterItem] of afterMap) {
    const beforeItem = beforeMap.get(key);
    if (!beforeItem) {
      added.push(afterItem);
    } else {
      const changes = computeCrawlFieldChanges(beforeItem, afterItem);
      if (changes.length > 0) {
        modified.push({
          url: key,
          before: beforeItem,
          after: afterItem,
          changes,
        });
      } else {
        unchanged.push(afterItem);
      }
    }
  }

  // Find removed items
  for (const [key, beforeItem] of beforeMap) {
    if (!afterMap.has(key)) {
      removed.push(beforeItem);
    }
  }

  return { added, removed, modified, unchanged };
}

/**
 * Compare two research results.
 *
 * @param before - Previous research result
 * @param after - Current research result
 * @returns Detailed diff result
 */
export function diffResearchResults(
  before: ResearchResultItem,
  after: ResearchResultItem,
): ResearchDiffResult {
  const result: ResearchDiffResult = {
    evidenceAdded: [],
    evidenceRemoved: [],
    evidenceModified: [],
    evidenceUnchanged: [],
    clustersAdded: [],
    clustersRemoved: [],
    clustersModified: [],
    citationsAdded: [],
    citationsRemoved: [],
  };

  // Compare summary
  if (before.summary !== after.summary) {
    result.summaryChanges = {
      oldValue: before.summary,
      newValue: after.summary,
    };
  }

  // Compare confidence
  if (before.confidence !== after.confidence) {
    result.confidenceChanges = {
      oldValue: before.confidence,
      newValue: after.confidence,
    };
  }

  // Compare evidence
  const beforeEvidence = before.evidence || [];
  const afterEvidence = after.evidence || [];
  const beforeEvidenceMap = new Map(
    beforeEvidence.map((e) => [computeEvidenceKey(e), e]),
  );
  const afterEvidenceMap = new Map(
    afterEvidence.map((e) => [computeEvidenceKey(e), e]),
  );

  for (const [key, afterItem] of afterEvidenceMap) {
    const beforeItem = beforeEvidenceMap.get(key);
    if (!beforeItem) {
      result.evidenceAdded.push(afterItem);
    } else {
      const changes = computeEvidenceFieldChanges(beforeItem, afterItem);
      if (changes.length > 0) {
        result.evidenceModified.push({
          url: key,
          before: beforeItem,
          after: afterItem,
          changes,
        });
      } else {
        result.evidenceUnchanged.push(afterItem);
      }
    }
  }

  for (const [key, beforeItem] of beforeEvidenceMap) {
    if (!afterEvidenceMap.has(key)) {
      result.evidenceRemoved.push(beforeItem);
    }
  }

  // Compare clusters
  const beforeClusters = before.clusters || [];
  const afterClusters = after.clusters || [];
  const beforeClusterMap = new Map(
    beforeClusters.map((c) => [computeClusterKey(c), c]),
  );
  const afterClusterMap = new Map(
    afterClusters.map((c) => [computeClusterKey(c), c]),
  );

  for (const [key, afterItem] of afterClusterMap) {
    const beforeItem = beforeClusterMap.get(key);
    if (!beforeItem) {
      result.clustersAdded.push(afterItem);
    } else {
      const changes = computeClusterFieldChanges(beforeItem, afterItem);
      if (changes.length > 0) {
        result.clustersModified.push({
          id: key,
          before: beforeItem,
          after: afterItem,
          changes,
        });
      }
    }
  }

  for (const [key, beforeItem] of beforeClusterMap) {
    if (!afterClusterMap.has(key)) {
      result.clustersRemoved.push(beforeItem);
    }
  }

  // Compare citations
  const beforeCitations = before.citations || [];
  const afterCitations = after.citations || [];
  const beforeCitationSet = new Set(beforeCitations.map(computeCitationKey));
  const afterCitationSet = new Set(afterCitations.map(computeCitationKey));

  for (const citation of afterCitations) {
    const key = computeCitationKey(citation);
    if (!beforeCitationSet.has(key)) {
      result.citationsAdded.push(citation);
    }
  }

  for (const citation of beforeCitations) {
    const key = computeCitationKey(citation);
    if (!afterCitationSet.has(key)) {
      result.citationsRemoved.push(citation);
    }
  }

  return result;
}

/**
 * Type guard for crawl result items.
 *
 * @param item - Result item to check
 * @returns True if item is a crawl result
 */
function isCrawlResultItem(item: ResultItem): item is CrawlResultItem {
  return "url" in item && "status" in item;
}

/**
 * Type guard for research result items.
 *
 * @param item - Result item to check
 * @returns True if item is a research result
 */
function isResearchResultItem(item: ResultItem): item is ResearchResultItem {
  return "summary" in item || "evidence" in item;
}

/**
 * Compare two sets of results (crawl or research).
 * Automatically detects result type and delegates to appropriate diff function.
 *
 * @param before - Previous results
 * @param after - Current results
 * @returns Diff result (crawl or research type)
 */
export function diffResults(
  before: ResultItem[],
  after: ResultItem[],
): CrawlDiffResult | ResearchDiffResult | null {
  if (before.length === 0 && after.length === 0) {
    return null;
  }

  // Detect result type from first non-null item
  const sampleBefore = before.find((item) => item != null);
  const sampleAfter = after.find((item) => item != null);
  const sample = sampleBefore || sampleAfter;

  if (!sample) return null;

  if (isCrawlResultItem(sample)) {
    return diffCrawlResults(
      before.filter(isCrawlResultItem),
      after.filter(isCrawlResultItem),
    );
  }

  if (isResearchResultItem(sample)) {
    // Research results are typically single items
    const beforeItem = (before.find(isResearchResultItem) ||
      {}) as ResearchResultItem;
    const afterItem = (after.find(isResearchResultItem) ||
      {}) as ResearchResultItem;
    return diffResearchResults(beforeItem, afterItem);
  }

  return null;
}

/**
 * Get summary statistics for a crawl diff.
 *
 * @param diff - Crawl diff result
 * @returns Summary counts
 */
export function getCrawlDiffStats(diff: CrawlDiffResult): {
  added: number;
  removed: number;
  modified: number;
  unchanged: number;
  total: number;
} {
  return {
    added: diff.added.length,
    removed: diff.removed.length,
    modified: diff.modified.length,
    unchanged: diff.unchanged.length,
    total:
      diff.added.length +
      diff.removed.length +
      diff.modified.length +
      diff.unchanged.length,
  };
}

/**
 * Get summary statistics for a research diff.
 *
 * @param diff - Research diff result
 * @returns Summary counts
 */
export function getResearchDiffStats(diff: ResearchDiffResult): {
  evidenceAdded: number;
  evidenceRemoved: number;
  evidenceModified: number;
  clustersAdded: number;
  clustersRemoved: number;
  clustersModified: number;
  citationsAdded: number;
  citationsRemoved: number;
  summaryChanged: boolean;
  confidenceChanged: boolean;
} {
  return {
    evidenceAdded: diff.evidenceAdded.length,
    evidenceRemoved: diff.evidenceRemoved.length,
    evidenceModified: diff.evidenceModified.length,
    clustersAdded: diff.clustersAdded.length,
    clustersRemoved: diff.clustersRemoved.length,
    clustersModified: diff.clustersModified.length,
    citationsAdded: diff.citationsAdded.length,
    citationsRemoved: diff.citationsRemoved.length,
    summaryChanged: !!diff.summaryChanges,
    confidenceChanged: !!diff.confidenceChanges,
  };
}
